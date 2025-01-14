package ui

import (
	"archive/zip"
	"fmt"
	"github.com/koho/frpmgr/pkg/config"
	"github.com/koho/frpmgr/pkg/consts"
	"github.com/koho/frpmgr/pkg/util"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/thoas/go-funk"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ConfView struct {
	*walk.Composite

	// List view
	listView     *walk.TableView
	lsEditAction *walk.Action
	model        *SortedListModel

	// Toolbar view
	toolbar        *walk.ToolBar
	tbAddAction    *walk.Action
	tbDeleteAction *walk.Action
	tbExportAction *walk.Action
}

var cachedListViewIconsForWidthAndState = make(map[widthAndState]*walk.Bitmap)

func NewConfView() *ConfView {
	v := new(ConfView)
	v.model = NewSortedListModel(confList)
	return v
}

func (cv *ConfView) View() Widget {
	return Composite{
		StretchFactor: 1,
		AssignTo:      &cv.Composite,
		Layout:        VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			TableView{
				AssignTo:            &cv.listView,
				LastColumnStretched: true,
				HeaderHidden:        true,
				Columns:             []TableViewColumn{{DataMember: "Name"}},
				Model:               cv.model,
				ContextMenuItems: []MenuItem{
					Action{AssignTo: &cv.lsEditAction, Text: "编辑配置", Enabled: Bind("conf.Selected"), OnTriggered: cv.editCurrent},
					Action{Text: "打开配置文件", Enabled: Bind("conf.Selected"), OnTriggered: cv.onOpen},
					Separator{},
					Action{Text: "创建新配置", OnTriggered: cv.editNew},
					Action{Text: "从文件导入配置", OnTriggered: cv.onImport},
					Action{Text: "导出所有配置 (ZIP 压缩包)", Enabled: Bind("conf.Selected"), OnTriggered: cv.onExport},
					Separator{},
					Action{Text: "删除配置", Enabled: Bind("conf.Selected"), OnTriggered: cv.onDelete},
				},
				StyleCell: func(style *walk.CellStyle) {
					row := style.Row()
					if row < 0 || row >= len(cv.model.items) {
						return
					}
					conf := cv.model.items[row]
					margin := cv.listView.IntFrom96DPI(1)
					bitmapWidth := cv.listView.IntFrom96DPI(16)
					cacheKey := widthAndState{bitmapWidth, conf.State}
					if cacheValue, ok := cachedListViewIconsForWidthAndState[cacheKey]; ok {
						style.Image = cacheValue
						return
					}
					bitmap, err := walk.NewBitmapWithTransparentPixelsForDPI(walk.Size{bitmapWidth, bitmapWidth}, cv.listView.DPI())
					if err != nil {
						return
					}
					canvas, err := walk.NewCanvasFromImage(bitmap)
					if err != nil {
						return
					}
					bounds := walk.Rectangle{X: margin, Y: margin, Height: bitmapWidth - 2*margin, Width: bitmapWidth - 2*margin}
					err = canvas.DrawImageStretchedPixels(iconForState(conf.State, 14), bounds)
					canvas.Dispose()
					if err != nil {
						return
					}
					cachedListViewIconsForWidthAndState[cacheKey] = bitmap
					style.Image = bitmap
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true, SpacingZero: true},
				Children: []Widget{
					ToolBar{
						AssignTo:    &cv.toolbar,
						ButtonStyle: ToolBarButtonImageBeforeText,
						Orientation: Horizontal,
						Items: []MenuItem{
							Menu{
								OnTriggered: cv.editNew,
								Text:        "新建配置",
								Image:       loadSysIcon("shell32", consts.IconNewConf, 16),
								Items: []MenuItem{
									Action{
										AssignTo:    &cv.tbAddAction,
										Text:        "创建新配置",
										Image:       loadSysIcon("shell32", consts.IconCreate, 16),
										OnTriggered: cv.editNew,
									},
									Action{
										Text:        "从文件导入配置",
										Image:       loadSysIcon("shell32", consts.IconImport, 16),
										OnTriggered: cv.onImport,
									},
								},
							},
							Separator{},
							Action{
								Enabled:     Bind("conf.Selected"),
								AssignTo:    &cv.tbDeleteAction,
								Image:       loadSysIcon("shell32", consts.IconDelete, 16),
								OnTriggered: cv.onDelete,
							},
							Separator{},
							Action{
								Enabled:     Bind("conf.Selected"),
								AssignTo:    &cv.tbExportAction,
								Image:       loadSysIcon("imageres", consts.IconExport, 16),
								OnTriggered: cv.onExport,
							},
						},
					},
				},
			},
		},
	}
}

func (cv *ConfView) OnCreate() {
	// Setup config list view
	cv.listView.ItemActivated().Attach(cv.editCurrent)
	cv.lsEditAction.SetDefault(true)
	cv.listView.CurrentIndexChanged().Attach(func() {
		if idx := cv.listView.CurrentIndex(); idx >= 0 && idx < len(cv.model.items) {
			setCurrentConf(cv.model.items[idx])
		} else {
			setCurrentConf(nil)
		}
	})
	// Setup toolbar
	cv.tbAddAction.SetDefault(true)
	cv.tbDeleteAction.SetToolTip("删除配置")
	cv.tbExportAction.SetToolTip("导出所有配置 (ZIP 压缩包)")
	cv.toolbar.ApplyDPI(cv.DPI())
	cv.fixWidthToToolbarWidth()
	cv.toolbar.SizeChanged().Attach(cv.fixWidthToToolbarWidth)
}

func (cv *ConfView) editCurrent() {
	cv.onEditConf(getCurrentConf())
}

func (cv *ConfView) editNew() {
	cv.onEditConf(nil)
}

func (cv *ConfView) fixWidthToToolbarWidth() {
	toolbarWidth := cv.toolbar.SizeHint().Width
	cv.SetMinMaxSizePixels(walk.Size{toolbarWidth, 0}, walk.Size{toolbarWidth, 0})
}

func (cv *ConfView) onEditConf(conf *Conf) {
	dlg := NewEditClientDialog(conf)
	if dlg == nil {
		return
	}
	if res, _ := dlg.Run(cv.Form()); res == walk.DlgCmdOK {
		if conf == nil {
			// Created new config
			// The list is resorted, we should select by name
			cv.reset(dlg.Conf.Name)
		} else {
			cv.listView.Invalidate()
			// Reset current conf
			confDB.Reset()
		}
		// Commit the config
		commitConf(dlg.Conf, dlg.ShouldRestart)
	}
}

func (cv *ConfView) onImport() {
	dlg := walk.FileDialog{
		Filter: consts.FilterConfig + consts.FilterAllFiles,
		Title:  "从文件导入配置",
	}

	if ok, _ := dlg.ShowOpenMultiple(cv.Form()); !ok {
		return
	}
	for _, path := range dlg.FilePaths {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".ini":
			newPath := filepath.Base(path)
			if _, err := os.Stat(newPath); err == nil {
				baseName, _ := util.SplitExt(newPath)
				showWarningMessage(cv.Form(), "错误", fmt.Sprintf("无法导入配置: 另一个同名的配置「%s」已存在", baseName))
				continue
			}
			if _, err := util.CopyFile(path, newPath); err != nil {
				showErrorMessage(cv.Form(), "错误", "复制文件时出现错误")
				continue
			}
			conf, err := config.UnmarshalClientConfFromIni(newPath)
			if err != nil {
				showError(err, cv.Form())
				continue
			}
			addConf(NewConf(newPath, conf))
		case ".zip":
			imported := 0
			total, copied := cv.unzipFiles(path)
			for _, cp := range copied {
				if conf, err := config.UnmarshalClientConfFromIni(cp); err == nil {
					addConf(NewConf(cp, conf))
					imported++
				}
			}
			if total != imported {
				showWarningMessage(cv.Form(), "导入配置", fmt.Sprintf("导入了 %d 个配置文件中的 %d 个", total, imported))
			}
		}
	}
	// Reselect the current config after refreshing list view
	if conf := getCurrentConf(); conf != nil {
		cv.reset(conf.Name)
	} else {
		cv.Invalidate()
	}
}

func (cv *ConfView) unzipFiles(path string) (total int, copied []string) {
	unzip := func(file *zip.File) error {
		fr, err := file.Open()
		if err != nil {
			return err
		}
		defer fr.Close()
		fw, err := os.OpenFile(file.Name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer fw.Close()
		if _, err = io.Copy(fw, fr); err != nil {
			return err
		}
		return nil
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		showErrorMessage(cv.Form(), "导入错误", "读取压缩文件时出现错误")
		return
	}
	defer zr.Close()
	copied = make([]string, 0)
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(file.Name)) != ".ini" {
			continue
		}
		total++
		// Skip the existing config
		if _, err = os.Stat(file.Name); err == nil {
			continue
		}
		if err = unzip(file); err == nil {
			copied = append(copied, file.Name)
		}
	}
	return
}

func (cv *ConfView) onOpen() {
	if conf := getCurrentConf(); conf != nil {
		if path, err := filepath.Abs(conf.Path); err == nil {
			openPath(path)
		}
	}
}

func (cv *ConfView) onDelete() {
	if conf := getCurrentConf(); conf != nil {
		if walk.MsgBox(cv.Form(), fmt.Sprintf("删除配置「%s」", conf.Name),
			fmt.Sprintf("确定要删除配置「%s」吗? 此操作无法撤销。", conf.Name),
			walk.MsgBoxOKCancel|walk.MsgBoxIconWarning) == walk.DlgCmdCancel {
			return
		}
		// Fully delete config
		if err := conf.Delete(); err != nil {
			showError(err, cv.Form())
			return
		}
		cv.Invalidate()
	}
}

func (cv *ConfView) onExport() {
	dlg := walk.FileDialog{
		Filter: consts.FilterZip,
		Title:  "导出配置文件 (ZIP 压缩包)",
	}

	if ok, _ := dlg.ShowSave(cv.Form()); !ok {
		return
	}

	if !strings.HasSuffix(dlg.FilePath, ".zip") {
		dlg.FilePath += ".zip"
	}

	files := funk.Map(confList, func(conf *Conf) string {
		return conf.Path
	})
	if err := util.ZipFiles(dlg.FilePath, files.([]string)); err != nil {
		showError(err, cv.Form())
	}
}

// reset config listview with selected name
func (cv *ConfView) reset(selectName string) {
	// Make sure `sel` is a valid index
	sel := funk.MaxInt([]int{cv.listView.CurrentIndex(), 0})
	// Refresh the whole config list
	// The confList will be sorted
	cv.model = NewSortedListModel(confList)
	cv.listView.SetModel(cv.model)
	if selectName != "" {
		sel = funk.MaxInt([]int{funk.IndexOf(cv.model.items, func(conf *Conf) bool { return conf.Name == selectName }), 0})
	}
	// Make sure the final selected index is valid
	if selectIdx := funk.MinInt([]int{sel, len(cv.model.items) - 1}); selectIdx >= 0 {
		cv.listView.SetCurrentIndex(selectIdx)
	}
}

// Invalidate conf view with last selected index
func (cv *ConfView) Invalidate() {
	cv.reset("")
}
