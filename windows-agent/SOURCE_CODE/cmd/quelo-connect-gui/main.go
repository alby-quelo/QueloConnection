package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/nossh/nossh/internal/configfile"
	"github.com/nossh/nossh/internal/launcher"
	nosshclient "github.com/nossh/nossh/internal/client"
)

const (
	appName    = "Quelo Connect"
	appVersion = "0.1.0-beta"
	appAuthor  = "Alberto Frosio"
	appEmail   = "alby@gnumerica.org"
)

type appState struct {
	app            *gtk.Application
	win            *gtk.Window
	host           string
	port           string
	token          string
	defaultMachine string
	serverLabel    *gtk.Label
	machineCombo   *gtk.ComboBoxText
	machineGrid    *gtk.Grid
	userEntry      *gtk.Entry
	connectBtn     *gtk.Button
	resetBtn       *gtk.Button
	connecting     bool
}

func (s *appState) serverAddr() string {
	return configfile.Client{
		Host:  s.host,
		Port:  s.port,
		Token: s.token,
	}.ServerAddr()
}

func (s *appState) updateServerLabel() {
	if s.serverLabel == nil {
		return
	}
	addr := s.serverAddr()
	if addr == "" {
		s.serverLabel.SetMarkup("<span foreground=\"#888888\">(non configurato — Opzioni → Configura server)</span>")
		return
	}
	s.serverLabel.SetText(addr)
}

func main() {
	gtk.Init(&os.Args)

	app, err := gtk.ApplicationNew("org.gnumerica.quelo-connect", glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		fatal(err)
	}

	var state *appState

	app.Connect("activate", func() {
		if state == nil {
			cfg := configfile.LoadClient()
			state = &appState{
				app:            app,
				host:           cfg.Host,
				port:           cfg.Port,
				token:          cfg.Token,
				defaultMachine: cfg.Machine,
			}
			if state.port == "" {
				state.port = configfile.DefaultClientPort
			}
			if err := state.buildUI(cfg); err != nil {
				fatal(err)
			}
			app.AddWindow(state.win)
		}
		state.win.ShowAll()
		state.win.Present()
	})

	os.Exit(app.Run(os.Args))
}

func (s *appState) buildUI(cfg configfile.Client) error {
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return err
	}
	s.win = win
	win.SetTitle(appName)
	win.SetDefaultSize(420, 300)
	win.SetResizable(false)
	win.SetBorderWidth(12)
	win.Connect("destroy", func() {
		s.app.Quit()
	})
	setWindowIcon(win)

	outer, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		return err
	}
	win.Add(outer)

	topRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	if err != nil {
		return err
	}
	outer.PackStart(topRow, false, false, 0)

	title, _ := gtk.LabelNew("")
	title.SetMarkup("<b>" + glib.MarkupEscapeText(appName) + "</b>")
	title.SetHAlign(gtk.ALIGN_START)
	topRow.PackStart(title, true, true, 0)

	optionsBtn, err := gtk.MenuButtonNew()
	if err != nil {
		return err
	}
	optionsBtn.SetLabel("Opzioni")
	buildOptionsMenu(s, optionsBtn)
	topRow.PackEnd(optionsBtn, false, false, 0)

	grid, err := gtk.GridNew()
	if err != nil {
		return err
	}
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(8)
	outer.PackStart(grid, true, true, 0)

	lblServer, _ := gtk.LabelNew("Server ponte:")
	lblServer.SetHAlign(gtk.ALIGN_START)
	grid.Attach(lblServer, 0, 0, 1, 1)

	serverLabel, _ := gtk.LabelNew("")
	serverLabel.SetHAlign(gtk.ALIGN_START)
	serverLabel.SetLineWrap(true)
	s.serverLabel = serverLabel
	grid.Attach(serverLabel, 1, 0, 1, 1)

	lblMachine, _ := gtk.LabelNew("Macchina:")
	lblMachine.SetHAlign(gtk.ALIGN_START)
	grid.Attach(lblMachine, 0, 1, 1, 1)

	machineCombo, err := gtk.ComboBoxTextNewWithEntry()
	if err != nil {
		return err
	}
	s.machineCombo = machineCombo
	s.machineGrid = grid
	for _, name := range configfile.LoadSavedMachines(cfg.Machine) {
		machineCombo.AppendText(name)
	}
	if entry, err := machineCombo.GetEntry(); err == nil && entry != nil {
		entry.SetPlaceholderText("nome registrato sul ponte")
		entry.SetText(cfg.Machine)
	}
	grid.Attach(machineCombo, 1, 1, 1, 1)

	lblUser, _ := gtk.LabelNew("Username:")
	lblUser.SetHAlign(gtk.ALIGN_START)
	grid.Attach(lblUser, 0, 2, 1, 1)

	userEntry, err := gtk.EntryNew()
	if err != nil {
		return err
	}
	s.userEntry = userEntry
	userEntry.SetPlaceholderText("utente Linux remoto")
	userEntry.SetActivatesDefault(true)
	grid.Attach(userEntry, 1, 2, 1, 1)

	lblPass, _ := gtk.LabelNew("Password:")
	lblPass.SetHAlign(gtk.ALIGN_START)
	grid.Attach(lblPass, 0, 3, 1, 1)

	hintPass, _ := gtk.LabelNew("(nel terminale che si apre)")
	hintPass.SetHAlign(gtk.ALIGN_START)
	grid.Attach(hintPass, 1, 3, 1, 1)

	btnRow, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	if err != nil {
		return err
	}
	btnRow.SetMarginTop(8)

	connectBtn, err := gtk.ButtonNewWithLabel("Connetti")
	if err != nil {
		return err
	}
	s.connectBtn = connectBtn
	btnRow.PackStart(connectBtn, true, true, 0)

	resetBtn, err := gtk.ButtonNewWithLabel("Reset")
	if err != nil {
		return err
	}
	s.resetBtn = resetBtn
	btnRow.PackStart(resetBtn, false, false, 0)

	grid.Attach(btnRow, 0, 4, 2, 1)

	connectBtn.Connect("clicked", func() { s.doConnect() })
	resetBtn.Connect("clicked", func() { s.resetConfig() })
	userEntry.Connect("activate", func() { s.doConnect() })

	s.updateServerLabel()
	return nil
}

func setWindowIcon(win *gtk.Window) {
	theme, err := gtk.IconThemeGetDefault()
	if err != nil {
		return
	}
	for _, name := range []string{"quelo-connect-gui", "quelo-connect"} {
		pixbuf, err := theme.LoadIcon(name, 64, gtk.ICON_LOOKUP_FORCE_SIZE)
		if err != nil {
			continue
		}
		win.SetIcon(pixbuf)
		return
	}
}

func buildOptionsMenu(state *appState, btn *gtk.MenuButton) {
	menu, _ := gtk.MenuNew()

	cfgItem, _ := gtk.MenuItemNewWithLabel("Configura server")
	cfgItem.Connect("activate", func() { state.configureServer() })
	menu.Append(cfgItem)

	addItem, _ := gtk.MenuItemNewWithLabel("Aggiungi macchina")
	addItem.Connect("activate", func() { state.addMachine() })
	menu.Append(addItem)

	delItem, _ := gtk.MenuItemNewWithLabel("Cancella macchina")
	delItem.Connect("activate", func() { state.deleteMachine() })
	menu.Append(delItem)

	saveItem, _ := gtk.MenuItemNewWithLabel("Salva configurazione")
	saveItem.Connect("activate", func() { state.saveConfig() })
	menu.Append(saveItem)

	resetItem, _ := gtk.MenuItemNewWithLabel("Reset configurazione")
	resetItem.Connect("activate", func() { state.resetConfig() })
	menu.Append(resetItem)

	helpItem, _ := gtk.MenuItemNewWithLabel("Help")
	helpItem.Connect("activate", func() { showInfo(state.win, gtk.MESSAGE_INFO, helpText()) })
	menu.Append(helpItem)

	aboutItem, _ := gtk.MenuItemNewWithLabel("About")
	aboutItem.Connect("activate", func() { showInfo(state.win, gtk.MESSAGE_INFO, aboutText()) })
	menu.Append(aboutItem)

	menu.ShowAll()
	btn.SetPopup(menu)
}

func (s *appState) machineName() string {
	if s.machineCombo == nil {
		return ""
	}
	if entry, err := s.machineCombo.GetEntry(); err == nil && entry != nil {
		text, _ := entry.GetText()
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(s.machineCombo.GetActiveText())
}

func (s *appState) saveConfig() {
	if err := s.persistConfig(); err != nil {
		showError(s.win, err.Error())
		return
	}
	showInfo(s.win, gtk.MESSAGE_INFO, "Configurazione salvata in client.conf")
}

func (s *appState) persistConfig() error {
	machine := s.machineName()
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
	dialog, err := gtk.DialogNew()
	if err != nil {
		showError(s.win, err.Error())
		return
	}
	dialog.SetTitle("Aggiungi macchina")
	dialog.SetTransientFor(s.win)
	dialog.SetModal(true)
	dialog.AddButton("Annulla", gtk.RESPONSE_CANCEL)
	dialog.AddButton("OK", gtk.RESPONSE_OK)

	content, err := dialog.GetContentArea()
	if err != nil {
		dialog.Destroy()
		return
	}

	box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	if err != nil {
		dialog.Destroy()
		return
	}
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	label, _ := gtk.LabelNew("Nome macchina registrata sul server ponte:")
	label.SetHAlign(gtk.ALIGN_START)
	box.PackStart(label, false, false, 0)

	entry, err := gtk.EntryNew()
	if err != nil {
		dialog.Destroy()
		return
	}
	entry.SetPlaceholderText("es. ufficio-server")
	entry.SetActivatesDefault(true)
	box.PackStart(entry, false, false, 0)
	content.Add(box)

	dialog.ShowAll()
	entry.GrabFocus()

	if dialog.Run() != gtk.RESPONSE_OK {
		dialog.Destroy()
		return
	}
	name, _ := entry.GetText()
	name = strings.TrimSpace(name)
	dialog.Destroy()

	if name == "" {
		return
	}

	if configfile.HasSavedMachine(name) {
		showInfo(s.win, gtk.MESSAGE_INFO, "Macchina già in elenco.")
		return
	}

	if s.serverAddr() == "" {
		showError(s.win, "Configura prima il server ponte (Opzioni → Configura server).")
		return
	}

	if err := nosshclient.CheckMachine(s.serverAddr(), name); err != nil {
		showError(s.win, err.Error())
		return
	}

	if err := configfile.SaveMachine(name); err != nil {
		showError(s.win, err.Error())
		return
	}

	s.machineCombo.AppendText(name)
	if comboEntry, err := s.machineCombo.GetEntry(); err == nil && comboEntry != nil {
		comboEntry.SetText(name)
	}
	showInfo(s.win, gtk.MESSAGE_INFO, fmt.Sprintf("Macchina %q aggiunta all'elenco.", name))
}

func (s *appState) deleteMachine() {
	names := configfile.ListSavedMachines()
	if len(names) == 0 {
		showInfo(s.win, gtk.MESSAGE_INFO, "Nessuna macchina salvata nell'elenco.")
		return
	}

	dialog, err := gtk.DialogNew()
	if err != nil {
		showError(s.win, err.Error())
		return
	}
	dialog.SetTitle("Cancella macchina")
	dialog.SetTransientFor(s.win)
	dialog.SetModal(true)
	dialog.AddButton("Annulla", gtk.RESPONSE_CANCEL)
	dialog.AddButton("OK", gtk.RESPONSE_OK)

	content, err := dialog.GetContentArea()
	if err != nil {
		dialog.Destroy()
		return
	}

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	box.SetMarginTop(8)
	box.SetMarginBottom(8)

	label, _ := gtk.LabelNew("Seleziona la macchina da rimuovere dall'elenco:")
	label.SetHAlign(gtk.ALIGN_START)
	box.PackStart(label, false, false, 0)

	combo, err := gtk.ComboBoxTextNew()
	if err != nil {
		dialog.Destroy()
		return
	}
	for _, n := range names {
		combo.AppendText(n)
	}
	combo.SetActive(0)
	box.PackStart(combo, false, false, 0)
	content.Add(box)

	dialog.ShowAll()

	if dialog.Run() != gtk.RESPONSE_OK {
		dialog.Destroy()
		return
	}
	name := strings.TrimSpace(combo.GetActiveText())
	dialog.Destroy()

	if name == "" {
		return
	}

	if err := configfile.DeleteSavedMachine(name); err != nil {
		showError(s.win, err.Error())
		return
	}

	if name == s.defaultMachine || name == s.machineName() {
		s.defaultMachine = ""
		if entry, err := s.machineCombo.GetEntry(); err == nil && entry != nil {
			entry.SetText("")
		}
		_ = configfile.SaveClient(configfile.Client{
			Host:    s.host,
			Port:    s.port,
			Token:   s.token,
			Machine: "",
		})
	}

	s.rebuildMachineCombo("")
	showInfo(s.win, gtk.MESSAGE_INFO, fmt.Sprintf("Macchina %q rimossa dall'elenco.", name))
}

func (s *appState) rebuildMachineCombo(selectName string) {
	if s.machineGrid == nil || s.machineCombo == nil {
		return
	}
	s.machineGrid.Remove(s.machineCombo)

	combo, err := gtk.ComboBoxTextNewWithEntry()
	if err != nil {
		return
	}
	s.machineCombo = combo

	names := configfile.ListSavedMachines()
	for _, n := range names {
		combo.AppendText(n)
	}
	pick := selectName
	if pick == "" {
		pick = s.defaultMachine
	}
	if entry, err := combo.GetEntry(); err == nil && entry != nil {
		entry.SetPlaceholderText("nome registrato sul ponte")
		if pick != "" {
			entry.SetText(pick)
		} else {
			entry.SetText("")
		}
	}
	s.machineGrid.Attach(combo, 1, 1, 1, 1)
	combo.ShowAll()
}

func (s *appState) configureServer() {
	dialog, err := gtk.DialogNew()
	if err != nil {
		showError(s.win, err.Error())
		return
	}
	dialog.SetTitle("Configura server")
	dialog.SetTransientFor(s.win)
	dialog.SetModal(true)
	dialog.AddButton("Annulla", gtk.RESPONSE_CANCEL)
	dialog.AddButton("Salva", gtk.RESPONSE_OK)

	content, err := dialog.GetContentArea()
	if err != nil {
		dialog.Destroy()
		return
	}

	grid, _ := gtk.GridNew()
	grid.SetRowSpacing(8)
	grid.SetColumnSpacing(8)
	grid.SetMarginStart(12)
	grid.SetMarginEnd(12)
	grid.SetMarginTop(8)
	grid.SetMarginBottom(8)

	lblHost, _ := gtk.LabelNew("IP server ponte:")
	grid.Attach(lblHost, 0, 0, 1, 1)
	hostEntry, _ := gtk.EntryNew()
	hostEntry.SetText(s.host)
	grid.Attach(hostEntry, 1, 0, 1, 1)

	lblPort, _ := gtk.LabelNew("Porta:")
	grid.Attach(lblPort, 0, 1, 1, 1)
	portEntry, _ := gtk.EntryNew()
	port := s.port
	if port == "" {
		port = configfile.DefaultClientPort
	}
	portEntry.SetText(port)
	grid.Attach(portEntry, 1, 1, 1, 1)

	lblToken, _ := gtk.LabelNew("Token:")
	grid.Attach(lblToken, 0, 2, 1, 1)
	tokenEntry, _ := gtk.EntryNew()
	tokenEntry.SetText(s.token)
	grid.Attach(tokenEntry, 1, 2, 1, 1)

	content.Add(grid)
	dialog.ShowAll()

	if dialog.Run() != gtk.RESPONSE_OK {
		dialog.Destroy()
		return
	}
	host, _ := hostEntry.GetText()
	portVal, _ := portEntry.GetText()
	token, _ := tokenEntry.GetText()
	dialog.Destroy()

	host = strings.TrimSpace(host)
	if host == "" {
		showError(s.win, "Inserisci l'IP del server ponte.")
		return
	}
	portVal = strings.TrimSpace(portVal)
	if portVal == "" {
		portVal = configfile.DefaultClientPort
	}

	s.host = host
	s.port = portVal
	s.token = strings.TrimSpace(token)
	if err := configfile.SaveClient(configfile.Client{
		Host:    s.host,
		Port:    s.port,
		Token:   s.token,
		Machine: s.machineName(),
	}); err != nil {
		showError(s.win, err.Error())
		return
	}
	s.updateServerLabel()
	showInfo(s.win, gtk.MESSAGE_INFO, "Server salvato in client.conf")
}

func (s *appState) resetConfig() {
	dialog := gtk.MessageDialogNew(s.win, gtk.DIALOG_MODAL, gtk.MESSAGE_WARNING, gtk.BUTTONS_YES_NO,
		"Vuoi cancellare tutta la configurazione (server, token, macchine salvate)?")
	if dialog.Run() != gtk.RESPONSE_YES {
		dialog.Destroy()
		return
	}
	dialog.Destroy()

	if err := configfile.ResetClient(); err != nil {
		showError(s.win, err.Error())
		return
	}
	s.host = ""
	s.port = configfile.DefaultClientPort
	s.token = ""
	s.defaultMachine = ""
	if s.userEntry != nil {
		s.userEntry.SetText("")
	}
	s.rebuildMachineCombo("")
	s.updateServerLabel()
	showInfo(s.win, gtk.MESSAGE_INFO, "Configurazione azzerata.")
}

func (s *appState) doConnect() {
	if s.connecting {
		return
	}

	machine := s.machineName()
	user := ""
	if s.userEntry != nil {
		user, _ = s.userEntry.GetText()
	}
	user = strings.TrimSpace(user)

	server := s.serverAddr()
	if server == "" {
		showError(s.win, "Configura il server ponte da Opzioni → Configura server.")
		return
	}
	if machine == "" {
		showError(s.win, "Inserisci il nome della macchina.")
		return
	}
	if user == "" {
		showError(s.win, "Inserisci lo username Linux.")
		return
	}

	if err := configfile.SaveClient(configfile.Client{
		Host:    s.host,
		Port:    s.port,
		Token:   s.token,
		Machine: machine,
	}); err != nil {
		showError(s.win, err.Error())
		return
	}
	s.defaultMachine = machine

	nossh, err := launcher.NosshPath()
	if err != nil {
		showError(s.win, err.Error())
		return
	}

	s.setConnecting(true)

	proc, err := launcher.StartNosshConnect(nossh, server, machine, user)
	if err != nil {
		s.setConnecting(false)
		showError(s.win, err.Error())
		return
	}

	s.win.Hide()

	go func() {
		_ = proc.Wait()
		glib.IdleAdd(func() {
			s.onTerminalClosed()
		})
	}()
}

func (s *appState) onTerminalClosed() {
	if s.win == nil {
		s.setConnecting(false)
		return
	}
	s.setConnecting(false)
	s.win.ShowAll()
	s.win.Present()
}

func (s *appState) setConnecting(active bool) {
	s.connecting = active
	if s.connectBtn != nil {
		s.connectBtn.SetSensitive(!active)
	}
	if s.resetBtn != nil {
		s.resetBtn.SetSensitive(!active)
	}
}

func helpText() string {
	return `Uso di Quelo Connect

1. Opzioni → Configura server: IP ponte, porta (default 7000) e token.

2. Inserisci il nome della macchina registrata sul server ponte (stato active).

3. Inserisci lo username Linux sulla macchina remota.

4. Clicca Connetti: si apre il terminale per la password SSH.
   La finestra si nasconde; alla chiusura del terminale torna visibile.

Reset: cancella server, token e elenco macchine.

Richiede nossh nel PATH e ~/.config/nossh/client.conf.`
}

func aboutText() string {
	return fmt.Sprintf(`%s
Versione %s

Client grafico per SSH via server ponte nossh.

Autore: %s
%s

Licenza: uso libero NON commerciale.
Uso commerciale vietato senza autorizzazione scritta.`, appName, appVersion, appAuthor, appEmail)
}

func showError(parent *gtk.Window, msg string) {
	showInfo(parent, gtk.MESSAGE_ERROR, msg)
}

func showInfo(parent *gtk.Window, msgType gtk.MessageType, msg string) {
	dialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, msgType, gtk.BUTTONS_OK, "%s", msg)
	dialog.Run()
	dialog.Destroy()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
