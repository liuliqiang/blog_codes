package abstruct_factory

type Windows interface {
	SetWidth(w int)
	SetHeight(h int)
	SetButton(x, y int, btn Button)
}

type Button interface {
	SetText(text string)
}

type GUIFramework interface {
	NewWindows() Windows
	NewButton() Button
}

type MFCFramework struct {
}
type MFCWindows struct {
}

func (M *MFCWindows) SetButton(x, y int, btn Button) {
	panic("implement me")
}

type MFCButton struct {
}

func (M *MFCFramework) NewWindows() Windows {
	return &MFCWindows{}
}

func (M *MFCFramework) NewButton() Button {
	return &MFCButton{}
}

func (M *MFCButton) SetText(text string) {
	panic("implement me")
}

func (M *MFCWindows) SetWidth(w int) {
	panic("implement me")
}

func (M *MFCWindows) SetHeight(h int) {
	panic("implement me")
}

type QtFramework struct {
}
type QtWindows struct {
}

func (q *QtWindows) SetButton(x, y int, btn Button) {
	panic("implement me")
}

type QtButton struct {
}

func (q *QtFramework) NewWindows() Windows {
	return &QtWindows{}
}

func (q *QtFramework) NewButton() Button {
	return &QtButton{}
}

func (q *QtButton) SetText(text string) {
	panic("implement me")
}

func (q *QtWindows) SetWidth(w int) {
	panic("implement me")
}

func (q *QtWindows) SetHeight(h int) {
	panic("implement me")
}

func main() {
	GuiFramework := MFCFramework{}
	GuiFramework := QtFramework{}

	windows := GuiFramework.NewWindows()
	btn := GuiFramework.NewButton()

	windows.SetButton(10, 10, btn)
}
