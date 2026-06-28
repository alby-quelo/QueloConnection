//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"strings"

	"github.com/tailscale/walk"
	. "github.com/tailscale/walk/declarative"
	"github.com/nossh/nossh/internal/configfile"
	"github.com/nossh/nossh/internal/launcher"
	nosshclient "github.com/nossh/nossh/internal/client"
)

//go:embed icon.png
var iconPNG []byte

const (
	appName    = "Quelo Connect"
	appVersion = "0.1.0-beta"
	appAuthor  = "Alberto Frosio"
	appEmail   = "alby@gnumerica.org"
)

type appState struct {
	mw             *walk.MainWindow
	machineEdit    *walk.LineEdit
	machineCombo   *walk.ComboBox
	userEdit       *walk.LineEdit
	connectBtn     *walk.PushButton
	resetBtn       *walk.PushButton
	serverLabel    *walk.Label
	host           string
	port           string
	token          string
	defaultMachine string
	connecting     bool
}

func main() {
	app, err := walk.InitApp()
	if err != nil {
		log.Fatal(err)
	}

	cfg := configfile.LoadClient()
	s := &appState{
		host:           cfg.Host,
		port:           cfg.Port,
		token:          cfg.Token,
		defaultMachine: cfg.Machine,
	}
	if s.port == "" {
		s.port = configfile.DefaultClientPort
	}

	if err := s.buildUI(cfg); err != nil {
		log.Fatal(err)
	}
	if ic, err := loadAppIcon(); err == nil {
		s.mw.SetIcon(ic)
	}
	s.updateServerLabel()

	app.Run()
}

func loadAppIcon() (*walk.Icon, error) {
	img, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return nil, err
	}
	return walk.NewIconFromImage(img)
}

func (s *appState) serverAddr() string {
	return configfile.Client{
		Host: s.host,
		Port: s.port,
		Token: s.token,
	}.ServerAddr()
}

func (s *appState) updateServerLabel() {
	if s.serverLabel == nil {
		return
	}
	addr := s.serverAddr()
	if addr == "" {
		s.serverLabel.SetText("(non configurato — usa Opzioni → Configura server)")
		return
	}
	s.serverLabel.SetText(addr)
}

func (s *appState) buildUI(cfg configfile.Client) error {
	saved := configfile.LoadSavedMachines(cfg.Machine)

	return MainWindow{
		AssignTo:        &s.mw,
		Title:           appName,
		MinSize:         Size{Width: 420, Height: 300},
		MaxSize:         Size{Width: 420, Height: 300},
		DisableMaximize: true,
		DisableResizing: true,
		Layout:          VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
		MenuItems: []MenuItem{
			Menu{
				Text: "Opzioni",
				Items: []MenuItem{
					Action{Text: "Configura server", OnTriggered: s.configureServer},
					Action{Text: "Aggiungi macchina", OnTriggered: s.addMachine},
					Action{Text: "Cancella macchina", OnTriggered: s.deleteMachine},
					Separator{},
					Action{Text: "Salva configurazione", OnTriggered: s.saveConfig},
					Separator{},
					Action{Text: "Help", OnTriggered: func() { s.showInfo(helpText()) }},
					Action{Text: "About", OnTriggered: func() { s.showInfo(aboutText()) }},
				},
			},
		},
		Children: []Widget{
			Label{Text: appName, Font: Font{Bold: true, PointSize: 11}},
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "Server ponte:"},
					Label{AssignTo: &s.serverLabel, Text: ""},
					Label{Text: "Macchina:"},
					Composite{
						Layout: HBox{Spacing: 4},
						Children: []Widget{
							LineEdit{AssignTo: &s.machineEdit, Text: cfg.Machine},
							ComboBox{
								AssignTo:              &s.machineCombo,
								Model:                 saved,
								MinSize:               Size{Width: 110, Height: 0},
								OnCurrentIndexChanged: s.onMachinePicked,
							},
						},
					},
					Label{Text: "Username:"},
					LineEdit{AssignTo: &s.userEdit},
					Label{Text: "Password:"},
					Label{Text: "(nel terminale che si apre)"},
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					PushButton{
						AssignTo:  &s.connectBtn,
						Text:      "Connetti",
						OnClicked: s.doConnect,
					},
					PushButton{
						AssignTo:  &s.resetBtn,
						Text:      "Reset",
						OnClicked: s.resetConfig,
					},
				},
			},
		},
	}.Create()
}

func (s *appState) onMachinePicked() {
	if s.machineCombo == nil || s.machineEdit == nil {
		return
	}
	if t := s.machineCombo.Text(); t != "" {
		s.machineEdit.SetText(t)
	}
}

func (s *appState) configureServer() {
	var dlg *walk.Dialog
	var acceptBtn, cancelBtn *walk.PushButton
	var hostEdit, portEdit, tokenEdit *walk.LineEdit
	var saved struct {
		host, port, token string
		ok                bool
	}

	port := s.port
	if port == "" {
		port = configfile.DefaultClientPort
	}

	accept := func() {
		if hostEdit != nil {
			saved.host = strings.TrimSpace(hostEdit.Text())
		}
		if portEdit != nil {
			saved.port = strings.TrimSpace(portEdit.Text())
		}
		if tokenEdit != nil {
			saved.token = strings.TrimSpace(tokenEdit.Text())
		}
		if saved.host == "" {
			walk.MsgBox(dlg, appName, "Inserisci l'IP del server ponte.", walk.MsgBoxIconError|walk.MsgBoxOK)
			return
		}
		if saved.port == "" {
			saved.port = configfile.DefaultClientPort
		}
		saved.ok = true
		dlg.Accept()
	}

	if err := (Dialog{
		AssignTo:      &dlg,
		Title:         "Configura server",
		DefaultButton: &acceptBtn,
		CancelButton:  &cancelBtn,
		MinSize:       Size{Width: 400, Height: 200},
		Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2, Spacing: 8},
				Children: []Widget{
					Label{Text: "IP server ponte:"},
					LineEdit{AssignTo: &hostEdit, Text: s.host},
					Label{Text: "Porta:"},
					LineEdit{AssignTo: &portEdit, Text: port},
					Label{Text: "Token:"},
					LineEdit{AssignTo: &tokenEdit, Text: s.token},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptBtn, Text: "Salva", OnClicked: accept},
					PushButton{AssignTo: &cancelBtn, Text: "Annulla", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}).Create(s.mw); err != nil {
		return
	}
	if dlg.Run() != walk.DlgCmdOK || !saved.ok {
		return
	}

	s.host = saved.host
	s.port = saved.port
	s.token = saved.token

	if err := s.persistConfig(); err != nil {
		s.showError(err.Error())
		return
	}
	s.updateServerLabel()
	s.showInfo("Server salvato in client.conf")
}

func (s *appState) resetConfig() {
	if walk.MsgBox(s.mw, appName,
		"Vuoi cancellare tutta la configurazione (server, token, macchine salvate)?",
		walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	if err := configfile.ResetClient(); err != nil {
		s.showError(err.Error())
		return
	}
	s.host = ""
	s.port = configfile.DefaultClientPort
	s.token = ""
	s.defaultMachine = ""
	if s.machineEdit != nil {
		s.machineEdit.SetText("")
	}
	if s.userEdit != nil {
		s.userEdit.SetText("")
	}
	s.refreshMachineCombo("")
	s.updateServerLabel()
	s.showInfo("Configurazione azzerata.")
}

func (s *appState) saveConfig() {
	if err := s.persistConfig(); err != nil {
		s.showError(err.Error())
		return
	}
	s.showInfo("Configurazione salvata in client.conf")
}

func (s *appState) persistConfig() error {
	machine := strings.TrimSpace(s.machineEdit.Text())
	if s.serverAddr() == "" {
		return fmt.Errorf("configura prima il server ponte (Opzioni → Configura server)")
	}
	s.defaultMachine = machine
	return configfile.SaveClient(configfile.Client{
		Host:    s.host,
		Port:    s.port,
		Token:   s.token,
		Machine: machine,
	})
}

func (s *appState) addMachine() {
	var dlg *walk.Dialog
	var acceptBtn, cancelBtn *walk.PushButton
	var nameEdit *walk.LineEdit
	var savedName string

	accept := func() {
		if nameEdit != nil {
			savedName = strings.TrimSpace(nameEdit.Text())
		}
		dlg.Accept()
	}

	if err := (Dialog{
		AssignTo:      &dlg,
		Title:         "Aggiungi macchina",
		DefaultButton: &acceptBtn,
		CancelButton:  &cancelBtn,
		MinSize:       Size{Width: 360, Height: 120},
		Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Label{Text: "Nome macchina registrata sul server ponte:"},
			LineEdit{AssignTo: &nameEdit},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptBtn, Text: "OK", OnClicked: accept},
					PushButton{AssignTo: &cancelBtn, Text: "Annulla", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}).Create(s.mw); err != nil {
		return
	}
	if dlg.Run() != walk.DlgCmdOK {
		return
	}

	name := savedName
	if name == "" {
		return
	}
	if configfile.HasSavedMachine(name) {
		s.showInfo("Macchina già in elenco.")
		return
	}
	server := s.serverAddr()
	if server == "" {
		s.showError("Configura prima il server ponte (Opzioni → Configura server).")
		return
	}
	if err := nosshclient.CheckMachine(server, name); err != nil {
		s.showError(err.Error())
		return
	}
	if err := configfile.SaveMachine(name); err != nil {
		s.showError(err.Error())
		return
	}
	s.refreshMachineCombo(name)
	s.machineEdit.SetText(name)
	s.showInfo(fmt.Sprintf("Macchina %q aggiunta all'elenco.", name))
}

func (s *appState) deleteMachine() {
	names := configfile.ListSavedMachines()
	if len(names) == 0 {
		s.showInfo("Nessuna macchina salvata nell'elenco.")
		return
	}

	var dlg *walk.Dialog
	var acceptBtn, cancelBtn *walk.PushButton
	var combo *walk.ComboBox
	var savedName string

	accept := func() {
		if combo != nil {
			savedName = strings.TrimSpace(combo.Text())
		}
		dlg.Accept()
	}

	if err := (Dialog{
		AssignTo:      &dlg,
		Title:         "Cancella macchina",
		DefaultButton: &acceptBtn,
		CancelButton:  &cancelBtn,
		MinSize:       Size{Width: 360, Height: 120},
		Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Label{Text: "Seleziona la macchina da rimuovere:"},
			ComboBox{AssignTo: &combo, Model: names},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &acceptBtn, Text: "OK", OnClicked: accept},
					PushButton{AssignTo: &cancelBtn, Text: "Annulla", OnClicked: func() { dlg.Cancel() }},
				},
			},
		},
	}).Create(s.mw); err != nil {
		return
	}
	if dlg.Run() != walk.DlgCmdOK {
		return
	}

	name := savedName
	if name == "" {
		return
	}
	if err := configfile.DeleteSavedMachine(name); err != nil {
		s.showError(err.Error())
		return
	}
	if name == s.defaultMachine || strings.TrimSpace(s.machineEdit.Text()) == name {
		s.defaultMachine = ""
		if s.machineEdit != nil {
			s.machineEdit.SetText("")
		}
		_ = configfile.SaveClient(configfile.Client{
			Host:    s.host,
			Port:    s.port,
			Token:   s.token,
			Machine: "",
		})
	}
	s.refreshMachineCombo("")
	s.showInfo(fmt.Sprintf("Macchina %q rimossa dall'elenco.", name))
}

func (s *appState) refreshMachineCombo(selectName string) {
	names := configfile.ListSavedMachines()
	s.machineCombo.SetModel(names)
	if selectName != "" {
		for i, n := range names {
			if n == selectName {
				s.machineCombo.SetCurrentIndex(i)
				break
			}
		}
		if s.machineEdit != nil {
			s.machineEdit.SetText(selectName)
		}
	} else if s.machineEdit != nil && s.defaultMachine == "" {
		s.machineEdit.SetText("")
	}
}

func (s *appState) doConnect() {
	if s.connecting {
		return
	}

	server := s.serverAddr()
	machine := strings.TrimSpace(s.machineEdit.Text())
	user := strings.TrimSpace(s.userEdit.Text())

	if server == "" {
		s.showError("Configura il server ponte da Opzioni → Configura server.")
		return
	}
	if machine == "" {
		s.showError("Inserisci il nome della macchina.")
		return
	}
	if user == "" {
		s.showError("Inserisci lo username Linux.")
		return
	}

	if err := configfile.SaveClient(configfile.Client{
		Host:    s.host,
		Port:    s.port,
		Token:   s.token,
		Machine: machine,
	}); err != nil {
		s.showError(err.Error())
		return
	}
	s.defaultMachine = machine

	nossh, err := launcher.NosshPath()
	if err != nil {
		s.showError(err.Error())
		return
	}

	s.setConnecting(true)
	proc, err := launcher.StartNosshConnect(nossh, server, machine, user)
	if err != nil {
		s.setConnecting(false)
		s.showError(err.Error())
		return
	}

	s.mw.SetVisible(false)

	go func() {
		_ = proc.Wait()
		s.mw.Synchronize(func() {
			s.onTerminalClosed()
		})
	}()
}

func (s *appState) onTerminalClosed() {
	s.setConnecting(false)
	s.mw.SetVisible(true)
	s.mw.Show()
}

func (s *appState) setConnecting(active bool) {
	s.connecting = active
	if s.connectBtn != nil {
		s.connectBtn.SetEnabled(!active)
	}
}

func (s *appState) showError(msg string) {
	walk.MsgBox(s.mw, appName, msg, walk.MsgBoxIconError|walk.MsgBoxOK)
}

func (s *appState) showInfo(msg string) {
	walk.MsgBox(s.mw, appName, msg, walk.MsgBoxIconInformation|walk.MsgBoxOK)
}

func helpText() string {
	return `Uso di Quelo Connect (Windows)

1. Opzioni → Configura server: IP ponte, porta (default 7000) e token.

2. Macchina: nome registrata sul ponte (stato active).

3. Username Linux sulla macchina remota.

4. Connetti: si apre CMD per la password SSH.
   La finestra si nasconde; alla chiusura del terminale torna visibile.

Reset: cancella server, token e elenco macchine.

Richiede nossh.exe nella stessa cartella e OpenSSH Client di Windows.`
}

func aboutText() string {
	return fmt.Sprintf(`%s
Versione %s

Client grafico portable per SSH via server ponte nossh.

Autore: %s
%s

Licenza: uso libero NON commerciale.
Uso commerciale vietato senza autorizzazione scritta.`, appName, appVersion, appAuthor, appEmail)
}
