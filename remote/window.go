package main

import (
	"syscall"
	"unsafe"

	"github.com/ying32/govcl/vcl/types/colors"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	_ "github.com/ying32/govcl/pkgs/winappres"
	"github.com/ying32/govcl/vcl"
)

var Form1 *TForm1
var atlDll = syscall.NewLazyDLL("atl.dll")
var AtlAxAttachControl = atlDll.NewProc("AtlAxAttachControl")

func main() {
	vcl.Application.Initialize()
	vcl.Application.CreateForm(&Form1)
	vcl.Application.Run()
}

type TForm1 struct {
	*vcl.TForm
	Rdp1 RdpPanel
}

type RdpPanel struct {
	*vcl.TPanel
	rdp *ole.IDispatch
}

func (f *TForm1) OnFormCreate(sender vcl.IObject) {
	f.SetCaption("windows远程桌面")
	f.SetBounds(10, 10, 1024, 800)
	f.Rdp1.Initrdp(f, 0, 0, 1024, 768, "192.168.2.28", "administrator", "pass")
}

func (rp *RdpPanel) Initrdp(f *TForm1, x, y, w, h int32, ip, username, pass string) {
	ole.CoInitialize(0)
	rp.TPanel = vcl.NewPanel(f)
	rp.SetParent(f)
	rp.SetBounds(x, y, w, h)
	rp.SetParentBackground(false)
	rp.SetColor(colors.ClRed)
	unknown, _ := oleutil.CreateObject("MsTscAx.MsTscAx.2")
	rp.rdp = unknown.MustQueryInterface(ole.IID_IDispatch)
	AtlAxAttachControl.Call(uintptr(unsafe.Pointer(&unknown.RawVTable)), rp.Handle(), 0)
	oleutil.PutProperty(rp.rdp, "server", ip)
	oleutil.PutProperty(rp.rdp, "username", username)
	set, _ := oleutil.GetProperty(rp.rdp, "AdvancedSettings")
	set.ToIDispatch().PutProperty("ClearTextPassword", pass)
	oleutil.MustCallMethod(rp.rdp, "connect")
	ole.CoUninitialize()
}
