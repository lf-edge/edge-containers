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

func (d DiskType) String() string {
	return [...]string{"Raw", "Vmdk", "Vhd", "ISO", "Qcow", "Qcow2", "Ova", "Vhdx"}[d]
}

type Disk struct {
	Path string
	Type DiskType
}

type Artifact struct {
	// Kernel path to the kernel file
	Kernel string
	// Initrd path to the initrd file
	Initrd string
	// Config path to the config
	Config string
	// Root path to the root disk and its type
	Root *Disk
	// Disks paths and types for additional disks
	Disks []*Disk
	// Other other items that did not have appropriate annotations
	Other []string
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
	Vhdx:  MimeTypeECIDiskVhdx,
}
var MimeToType = map[string]DiskType{
	MimeTypeECIDiskRaw:   Raw,
	MimeTypeECIDiskVhd:   Vhd,
	MimeTypeECIDiskVmdk:  Vmdk,
	MimeTypeECIDiskISO:   ISO,
	MimeTypeECIDiskQcow:  Qcow,
	MimeTypeECIDiskQcow2: Qcow2,
	MimeTypeECIDiskOva:   Ova,
	MimeTypeECIDiskVhdx:  Vhdx,
}
