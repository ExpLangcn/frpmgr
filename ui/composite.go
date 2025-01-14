package ui

import (
	"github.com/koho/frpmgr/pkg/config"
	"github.com/koho/frpmgr/pkg/consts"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// quickAddBinder is the view model of quick-add dialog
type quickAddBinder struct {
	RemotePort   string
	LocalAddr    string
	LocalPort    string
	LocalPortMin string
	LocalPortMax string
	Dir          string
}

// QuickAdd is the interface that must be implemented to build a quick-add dialog
type QuickAdd interface {
	// Run a new simple dialog to input few parameters
	Run(owner walk.Form) (int, error)
	// GetProxies returns the proxies generated by quick-add dialog
	GetProxies() []*config.Proxy
}

// NewBrowseLineEdit places a tool button at the tail of a LineEdit, and opens a file dialog when the button is clicked
func NewBrowseLineEdit(assignTo **walk.LineEdit, visible, enable, text Property, title, filter string, file bool) Composite {
	var editView *walk.LineEdit
	if assignTo == nil {
		assignTo = &editView
	}
	return Composite{
		Visible: visible,
		Layout:  HBox{MarginsZero: true, SpacingZero: false, Spacing: 3},
		Children: []Widget{
			LineEdit{Enabled: enable, AssignTo: assignTo, Text: text},
			ToolButton{Enabled: enable, Text: "...", MaxSize: Size{Width: 24}, OnClicked: func() {
				openFileDialog(*assignTo, title, filter, file)
			}},
		}}
}

// NewBasicDialog returns a dialog with given widgets and default buttons
func NewBasicDialog(assignTo **walk.Dialog, title string, icon Property, db DataBinder, yes func(), widgets ...Widget) Dialog {
	var w *walk.Dialog
	if assignTo == nil {
		assignTo = &w
	}
	if yes == nil {
		// Default handler for "yes" button
		yes = func() {
			if err := (*assignTo).DataBinder().Submit(); err == nil {
				(*assignTo).Accept()
			}
		}
	}
	var acceptPB, cancelPB *walk.PushButton
	dlg := Dialog{
		AssignTo:      assignTo,
		Icon:          icon,
		Title:         title,
		Layout:        VBox{},
		Font:          consts.TextRegular,
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder:    db,
		Children:      make([]Widget, 0),
	}
	dlg.Children = append(dlg.Children, widgets...)
	dlg.Children = append(dlg.Children, Composite{
		Layout: HBox{MarginsZero: true},
		Children: []Widget{
			HSpacer{},
			PushButton{Text: "确定", AssignTo: &acceptPB, OnClicked: yes},
			PushButton{Text: "取消", AssignTo: &cancelPB, OnClicked: func() { (*assignTo).Cancel() }},
		},
	})
	return dlg
}

// NewRadioButtonGroup returns a simple radio button group
func NewRadioButtonGroup(dataMember string, db *DataBinder, buttons []RadioButton) Composite {
	v := Composite{
		Layout: HBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			RadioButtonGroup{
				DataMember: dataMember,
				Buttons:    buttons,
			},
			HSpacer{},
		},
	}
	if db != nil {
		v.DataBinder = *db
	}
	return v
}
