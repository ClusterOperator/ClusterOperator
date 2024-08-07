package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ClusterOperator/ClusterOperator/pkg/constant"
	"github.com/ClusterOperator/ClusterOperator/pkg/controller/page"
	"github.com/ClusterOperator/ClusterOperator/pkg/db"
	"github.com/ClusterOperator/ClusterOperator/pkg/dto"
	"github.com/ClusterOperator/ClusterOperator/pkg/logger"
	"github.com/ClusterOperator/ClusterOperator/pkg/model"
	"github.com/ClusterOperator/ClusterOperator/pkg/repository"
	clusterUtil "github.com/ClusterOperator/ClusterOperator/pkg/util/cluster"
	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type CisService interface {
	Page(num, size int, clusterName string) (*page.Page, error)
	List(clusterName string) ([]dto.CisTask, error)
	Create(clusterName string, create *dto.CisTaskCreate) (*dto.CisTask, error)
	Delete(clusterName, id string) error
	Get(clusterName, id string) (*dto.CisTaskDetail, error)
}

type cisService struct {
	clusterRepo    repository.ClusterRepository
	clusterService ClusterService
}

func NewCisService() CisService {
	return &cisService{
		clusterRepo:    repository.NewClusterRepository(),
		clusterService: NewClusterService(),
	}
}

func (c *cisService) Get(clusterName, id string) (*dto.CisTaskDetail, error) {
	var cisTask model.CisTaskWithResult
	if err := db.DB.First(&cisTask, &model.CisTaskWithResult{CisTask: model.CisTask{ID: id}}).Error; err != nil {
		return nil, err
	}
	var report dto.CisReport
	if err := json.Unmarshal([]byte(cisTask.Result), &report); err != nil {
		return nil, err
	}

	cls, err := c.clusterService.Get(clusterName)
	if err != nil {
		return nil, err
	}
	d := &dto.CisTaskDetail{CisTaskWithResult: cisTask, CisReport: report}
	d.ClusterName = cls.Name
	d.ClusterVersion = cls.Version
	return d, nil
}

func (*cisService) Page(num, size int, clusterName string) (*page.Page, error) {
	var cluster model.Cluster
	if err := db.DB.Where("name = ?", clusterName).First(&cluster).Error; err != nil {
		return nil, err
	}
	p := page.Page{}
	var tasks []model.CisTask
	if err := db.DB.Model(&model.CisTask{}).
		Where("cluster_id = ?", cluster.ID).
		Count(&p.Total).
		Order("created_at desc").
		Offset((num - 1) * size).
		Limit(size).
		//Preload("Results").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	var dtos []dto.CisTask
	for _, task := range tasks {
		dtos = append(dtos, dto.CisTask{CisTask: task})
	}
	p.Items = dtos
	return &p, nil
}

const (
	CisTaskStatusCreating = "Creating"
	CisTaskStatusRunning  = "Running"
	CisTaskStatusSuccess  = "Success"
	CisTaskStatusFailed   = "Failed"
)

func (c *cisService) List(clusterName string) ([]dto.CisTask, error) {
	var cluster model.Cluster
	if err := db.DB.Where("name = ?", clusterName).First(&cluster).Error; err != nil {
		return nil, err
	}
	var tasks []model.CisTask
	if err := db.DB.
		Where("cluster_id = ?", cluster.ID).
		Preload("Results").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	var dtos []dto.CisTask
	for _, task := range tasks {
		dtos = append(dtos, dto.CisTask{CisTask: task})
	}
	return dtos, nil
}

func (c *cisService) Create(clusterName string, create *dto.CisTaskCreate) (*dto.CisTask, error) {
	cluster, err := c.clusterRepo.GetWithPreload(clusterName, []string{"SpecConf", "Secret", "Nodes", "Nodes.Host", "Nodes.Host.Credential"})
	if err != nil {
		return nil, fmt.Errorf("load cluster info failed, err: %v", err.Error())
	}
	var registery model.SystemRegistry
	if cluster.Architectures == constant.ArchAMD64 {
		if err := db.DB.Where("architecture = ?", constant.ArchitectureOfAMD64).First(&registery).Error; err != nil {
			return nil, fmt.Errorf("load image pull port of arm failed, err: %v", err.Error())
		}
	} else {
		if err := db.DB.Where("architecture = ?", constant.ArchitectureOfARM64).First(&registery).Error; err != nil {
			return nil, fmt.Errorf("load image pull port of arm failed, err: %v", err.Error())
		}
	}
	localRepoPort := registery.RegistryPort

	var clusterTasks []model.CisTask
	db.DB.Where("status = ? AND cluster_id = ?", constant.StatusRunning, cluster.ID).Find(&clusterTasks)
	if len(clusterTasks) > 0 {
		return nil, errors.New("CIS_TASK_ALREADY_RUNNING")
	}
	tx := db.DB.Begin()
	task := model.CisTask{
		ClusterID: cluster.ID,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Policy:    create.Policy,
		Status:    CisTaskStatusCreating,
	}
	err = tx.Create(&task).Error
	if err != nil {
		return nil, err
	}

	client, err := clusterUtil.NewClusterClient(&cluster)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	go Do(client, &task, localRepoPort)
	return &dto.CisTask{CisTask: task}, nil
}

func (c *cisService) Delete(clusterName, id string) error {
	cluster, err := c.clusterRepo.Get(clusterName)
	if err != nil {
		return err
	}
	if err := db.DB.Where("id = ? AND cluster_id = ?", id, cluster.ID).Delete(&model.CisTask{}).Error; err != nil {
		return err
	}
	return nil
}

const kubeBenchVersion = "v0.6.8"

func Do(client *kubernetes.Clientset, task *model.CisTask, port int) {
	taskWithResult := &model.CisTaskWithResult{CisTask: *task}

	taskWithResult.Status = CisTaskStatusRunning
	db.DB.Save(&taskWithResult)

	jobId := fmt.Sprintf("kube-bench-%s", uuid.NewV4().String())
	j := v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobId,
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "kube-bench"}},
				Spec: corev1.PodSpec{
					HostPID:       true,
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:    "kube-bench",
							Image:   fmt.Sprintf("%s:%d/kubeoperator/kube-bench:%s", constant.LocalRepositoryDomainName, port, kubeBenchVersion),
							Command: []string{"kube-bench", "--json"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "var-lib-etcd",
									MountPath: "/var/lib/etcd",
									ReadOnly:  true,
								},
								{
									Name:      "var-lib-kubelet",
									MountPath: "/var/lib/kubelet",
									ReadOnly:  true,
								},
								{
									Name:      "etc-systemd",
									MountPath: "/etc/systemd",
									ReadOnly:  true,
								},
								{
									Name:      "etc-kubernetes",
									MountPath: "/etc/kubernetes",
									ReadOnly:  true,
								},
								{
									Name:      "usr-bin",
									MountPath: "/usr/local/mount-from-host/bin",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "var-lib-etcd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/etcd",
								},
							},
						},
						{
							Name: "var-lib-kubelet",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet",
								},
							},
						},
						{
							Name: "etc-systemd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/systemd",
								},
							},
						},
						{
							Name: "etc-kubernetes",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/kubernetes",
								},
							},
						},
						{
							Name: "usr-bin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/usr/bin",
								},
							},
						},
					},
				},
			},
		},
	}
	if taskWithResult.Policy != "" && taskWithResult.Policy != "auto" {
		j.Spec.Template.Spec.Containers[0].Command = append(j.Spec.Template.Spec.Containers[0].Command, "--benchmark", taskWithResult.Policy)
	}

	resp, err := client.BatchV1().Jobs(constant.DefaultNamespace).Create(context.TODO(), &j, metav1.CreateOptions{})
	if err != nil {
		taskWithResult.Message = err.Error()
		taskWithResult.Status = CisTaskStatusFailed
		db.DB.Save(&taskWithResult)
		return
	}

	err = wait.Poll(5*time.Second, 5*time.Minute, func() (done bool, err error) {
		job, err := client.BatchV1().Jobs(constant.DefaultNamespace).Get(context.TODO(), resp.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		if job.Status.Succeeded > 0 {
			pds, err := client.CoreV1().Pods(constant.DefaultNamespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", resp.Name),
			})
			if err != nil {
				return true, err
			}
			for _, p := range pds.Items {
				if p.Status.Phase == corev1.PodSucceeded {
					r := client.CoreV1().Pods(constant.DefaultNamespace).GetLogs(p.Name, &corev1.PodLogOptions{})
					bs, err := r.DoRaw(context.TODO())
					if err != nil {
						return true, err
					}
					taskWithResult.Result = string(bs)
					var report dto.CisReport
					if err := json.Unmarshal(bs, &report); err != nil {
						return true, err
					}
					taskWithResult.TotalPass = report.Totals.TotalPass
					taskWithResult.TotalFail = report.Totals.TotalFail
					taskWithResult.TotalWarn = report.Totals.TotalWarn
					taskWithResult.TotalInfo = report.Totals.TotalInfo
					taskWithResult.Status = CisTaskStatusSuccess
					taskWithResult.EndTime = time.Now()
					db.DB.Save(&taskWithResult)
				}
			}
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		taskWithResult.Message = err.Error()
		taskWithResult.Status = CisTaskStatusFailed
		taskWithResult.EndTime = time.Now()
		db.DB.Save(&taskWithResult)
		return
	}
	err = client.BatchV1().Jobs(constant.DefaultNamespace).Delete(context.TODO(), resp.Name, metav1.DeleteOptions{})
	if err != nil {
		logger.Log.Error(err.Error())
		return
	}
}
