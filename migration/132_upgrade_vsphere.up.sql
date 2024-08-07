INSERT INTO `ko`.`ko_storage_provisioner_dic`(`id`, `name`, `version`, `architecture`, `vars`, `created_at`, `updated_at`) VALUES (
    UUID(), 'vsphere', 'v2.5.1', 'amd64', '{\"vsphere_csi_version\":\"v2.5.1\", \"govc_version\":\"v0.23.0\", \"vsphere_csi_attacher_version\":\"v3.4.0\", \"vsphere_csi_resizer_version\":\"v1.4.0\", \"vsphere_csi_livenessprobe_version\":\"v2.6.0\", \"vsphere_csi_provisioner_version\":\"v3.1.0\", \"vsphere_csi_snapshotter_version\":\"v5.0.1\", \"vsphere_csi_node_driver_registrar_version\":\"v2.5.0\"}', date_add(now(), interval 8 HOUR), date_add(now(), interval 8 HOUR));

UPDATE ko_cluster_manifest SET storage_vars='[{\"name\":\"external-ceph-block\",\"version\":\"v2.1.1-k8s1.11\"}, {\"name\":\"external-cephfs\",\"version\":\"v2.1.0-k8s1.11\"}, {\"name\":\"nfs\",\"version\":\"v3.1.0-k8s1.11\"}, {\"name\":\"vsphere\",\"version\":\"v2.5.1\"}, {\"name\":\"rook-ceph\",\"version\":\"v1.9.0\"}, {\"name\":\"oceanstor\",\"version\":\"v2.2.9\"}, {\"name\":\"cinder\",\"version\":\"v1.20.0\"}]';