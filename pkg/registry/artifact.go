package registry

type DiskType int

const (
	Raw DiskType = iota
	Vmdk
	Vhd
	ISO
	Qcow
	Qcow2
	Ova
	Vhdx
)

type Disk struct {
	Path string
	Type DiskType
}

type Artifact struct {
	Kernel string
	Initrd string
	Config string
	Legacy bool
	Root   *Disk
	Disks  []*Disk
}

var NameToType = map[string]DiskType{
	"raw":   Raw,
	"vmdk":  Vmdk,
	"vhd":   Vhd,
	"iso":   ISO,
	"qcow":  Qcow,
	"qcow2": Qcow2,
	"ova":   Ova,
	"vhdx":  Vhdx,
}
var TypeToMime = map[DiskType]string{
	Raw:   MimeTypeECIDiskRaw,
	Vhd:   MimeTypeECIDiskVhd,
	Vmdk:  MimeTypeECIDiskVmdk,
	ISO:   MimeTypeECIDiskISO,
	Qcow:  MimeTypeECIDiskQcow,
	Qcow2: MimeTypeECIDiskQcow2,
	Ova:   MimeTypeECIDiskOva,
	Vhdx:  MimeTypeECIDiskOva,
}
