package registry

const (
	MimeTypeECIConfig        = "application/vnd.lfedge.eci.config.v1+json"
	MimeTypeECIKernel        = "application/vnd.lfedge.eci.kernel.layer.v1+kernel"
	MimeTypeECIInitrd        = "application/vnd.lfedge.eci.initrd.layer.v1+cpio"
	MimeTypeECIDiskRaw       = "application/vnd.lfedge.disk.layer.v1+raw"
	MimeTypeECIDiskVhd       = "application/vnd.lfedge.disk.layer.v1+vhd"
	MimeTypeECIDiskVmdk      = "application/vnd.lfedge.disk.layer.v1+vmdk"
	MimeTypeECIDiskISO       = "application/vnd.lfedge.disk.layer.v1+iso"
	MimeTypeECIDiskQcow      = "application/vnd.lfedge.disk.layer.v1+qcow"
	MimeTypeECIDiskQcow2     = "application/vnd.lfedge.disk.layer.v1+qcow2"
	MimeTypeECIDiskOva       = "application/vnd.lfedge.disk.layer.v1+ova"
	MimeTypeECIDiskVhdx      = "application/vnd.lfedge.disk.layer.v1+vhdx"
	MimeTypeOCIImageConfig   = "application/vnd.oci.image.config.v1+json"
	MimeTypeOCIImageLayer    = "application/vnd.oci.image.layer.v1.tar"
	MimeTypeOCIImageManifest = "application/vnd.oci.image.manifest.v1+json"
	MimeTypeOCIImageIndex    = "application/vnd.oci.image.index.v1+json"
)

var allTypes = []string{
	MimeTypeECIConfig,
	MimeTypeECIKernel,
	MimeTypeECIInitrd,
	MimeTypeECIDiskRaw,
	MimeTypeECIDiskVhd,
	MimeTypeECIDiskVmdk,
	MimeTypeECIDiskISO,
	MimeTypeECIDiskQcow,
	MimeTypeECIDiskQcow2,
	MimeTypeECIDiskOva,
	MimeTypeECIDiskVhdx,
	MimeTypeOCIImageConfig,
	MimeTypeOCIImageLayer,
	MimeTypeOCIImageManifest,
	MimeTypeOCIImageIndex,
}

func AllMimeTypes() []string {
	return allTypes[:]
}

func GetLayerMediaType(actualType string, legacy bool) string {
	if legacy {
		return MimeTypeOCIImageLayer
	}
	return actualType
}
func GetConfigMediaType(actualType string, legacy bool) string {
	if legacy {
		return MimeTypeOCIImageConfig
	}
	return actualType
}
