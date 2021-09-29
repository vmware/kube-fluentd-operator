### Building the CRDs and regenerating the clients/informers/listers:

Detailed Guide for Kubebuilder and code-generator generation:
https://www.fatalerrors.org/a/writing-crd-by-mixing-kubeuilder-and-code-generator.html


#### Outline of Steps:

```bash
go mod init vdp.vmware.com/kube-fluentd-operator

kubebuilder init --plugins go/v3 --domain vdp.vmware.com --license apache2 --owner "VMware VDP Team" --repo vdp.vmware.com/kube-fluentd-operator --skip-go-version-check

kubebuilder edit --multigroup=true

kubebuilder create api --group logs --version v1beta1 --kind FluentdConfig
```

Follow the above tutorial to regenerate the clients, informers, listers with code-generator.
