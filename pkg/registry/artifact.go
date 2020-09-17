package registry

import "path"

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

// Source a source for an artifact component
type Source interface {
	// GetPath get path to a file, returns "" if no file
	GetPath() string
	// GetContent get the actual content if in memory, returns nil if in a file
	GetContent() []byte
	// GetName returns the target filename
	GetName() string
}

// FileSource implements a Source for a file
type FileSource struct {
	// Path path to the file source
	Path string
}

func (f *FileSource) GetPath() string {
	return f.Path
}
func (f *FileSource) GetContent() []byte {
	return nil
}
func (f *FileSource) GetName() string {
	return path.Base(f.Path)
}

// MemorySource implements a Source for raw data
type MemorySource struct {
	// Content the data
	Content []byte
	// Name name of file to save
	Name string
}

func (m *MemorySource) GetPath() string {
	return ""
}
func (m *MemorySource) GetContent() []byte {
	return m.Content
}
func (m *MemorySource) GetName() string {
	return m.Name
}

type Disk struct {
	Source Source
	Type   DiskType
}

type Artifact struct {
	// Kernel path to the kernel file
	Kernel Source
	// Initrd path to the initrd file
	Initrd Source
	// Config path to the config
	Config Source
	// Root path to the root disk and its type
	Root *Disk
	// Disks paths and types for additional disks
	Disks []*Disk
	// Other other items that did not have appropriate annotations
	Other []Source
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
