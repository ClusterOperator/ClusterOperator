update
    ko_cluster_manifest
set
    `is_active` = 0
where
    `name` = 'v1.22.8-ko1';

insert into
    `ko`.`ko_cluster_manifest`(
        `id`,
        `name`,
        `version`,
        `core_vars`,
        `network_vars`,
        `tool_vars`,
        `storage_vars`,
        `other_vars`,
        `created_at`,
        `updated_at`,
        `is_active`
    )
VALUES
    (
        UUID(),
        'v1.22.10-ko1',
        'v1.22.10',
        '[{\"name\":\"kubernetes\",\"version\":\"v1.22.10\"},{\"name\":\"docker\",\"version\":\"20.10.12\"},{\"name\":\"etcd\",\"version\":\"v3.5.2\"},{\"name\":\"containerd\",\"version\":\"1.6.0\"}]',
        '[{\"name\":\"calico\",\"version\":\"v3.21.4\"},{\"name\":\"flanneld\",\"version\":\"v0.15.1\"},{\"name\":\"cilium\",\"version\":\"v1.9.5\"}]',
        '[{"name":"gatekeeper","version":"v3.7.0"},{"name":"loki","version":"v2.1.0"},{"name":"kubeapps","version":"2.4.2"},{"name":"prometheus","version":"2.31.1"},{"name":"chartmuseum","version":"v0.12.0"},{"name":"registry","version":"v2.7.1"}, {"name":"grafana","version":"8.3.1"},{"name":"logging","version":"v7.6.2"}]',
        '[{\"name\":\"external-ceph-block\",\"version\":\"v2.1.1-k8s1.11\"}, {\"name\":\"external-cephfs\",\"version\":\"v2.1.0-k8s1.11\"}, {\"name\":\"nfs\",\"version\":\"v3.1.0-k8s1.11\"}, {\"name\":\"vsphere\",\"version\":\"v1.0.3\"}, {\"name\":\"rook-ceph\",\"version\":\"v1.9.0\"}, {\"name\":\"oceanstor\",\"version\":\"v2.2.9\"}, {\"name\":\"cinder\",\"version\":\"v1.20.0\"}]',
        '[{\"name\":\"coredns\",\"version\":\"1.8.4\"},{\"name\":\"dns-cache\",\"version\":\"1.17.0\"},{\"name\":\"traefik\",\"version\":\"v2.6.1\"},{\"name\":\"ingress-nginx\",\"version\":\"v1.1.1\"},{\"name\":\"metrics-server\",\"version\":\"v0.5.0\"},{\"name\":\"helm-v2\",\"version\":\"v2.17.0\"},{\"name\":\"helm-v3\",\"version\":\"v3.8.0\"}]',
        date_add(now(), interval 8 HOUR),
        date_add(now(), interval 8 HOUR),
        1
    );