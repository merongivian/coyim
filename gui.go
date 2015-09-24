// +build nocli

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/twstrike/coyim/config"
	"github.com/twstrike/coyim/gui"
	"github.com/twstrike/coyim/ui"
	"github.com/twstrike/coyim/xmpp"
	"github.com/twstrike/go-gtk/gdk"
	"github.com/twstrike/go-gtk/glib"
	"github.com/twstrike/go-gtk/gtk"
	"github.com/twstrike/otr3"
)

type gtkUI struct {
	roster  *gui.Roster
	session *Session
	window  *gtk.Window

	config    *config.Config
	connected bool
}

func (*gtkUI) RegisterCallback() xmpp.FormCallback {
	if *createAccount {
		return func(title, instructions string, fields []interface{}) error {
			//TODO: should open a registration window
			fmt.Println("TODO")
			return nil
		}
	}

	return nil
}

func (u *gtkUI) MessageReceived(from, timestamp string, encrypted bool, message []byte) {
	u.roster.MessageReceived(from, timestamp, encrypted, message)
}

func (u *gtkUI) NewOTRKeys(uid string, conversation *otr3.Conversation) {
	u.Info(fmt.Sprintf("TODO: notify new keys from %s", uid))
}

func (u *gtkUI) OTREnded(uid string) {
	//TODO: conversation ended
}

func (u *gtkUI) Info(m string) {
	fmt.Println(">>> INFO", m)
}

func (u *gtkUI) Warn(m string) {
	fmt.Println(">>> WARN", m)
}

func (u *gtkUI) Alert(m string) {
	fmt.Println(">>> ALERT", m)
}

func (u *gtkUI) Disconnected() {
	//TODO: remove everybody from the roster
	fmt.Println("TODO: Should disconnect the account")
}

func (u *gtkUI) Loop() {
	gtk.Init(&os.Args)
	gdk.ThreadsInit()

	gdk.ThreadsEnter()
	u.mainWindow()
	gtk.Main()
	gdk.ThreadsLeave()
}

func NewGTK() *gtkUI {
	return &gtkUI{}
}

func (u *gtkUI) mainWindow() {
	u.window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	u.roster = gui.NewRoster()
	u.roster.CheckEncrypted = u.checkEncrypted
	u.roster.SendMessage = u.sendMessage

	menubar := initMenuBar(u)
	vbox := gtk.NewVBox(false, 1)
	vbox.PackStart(menubar, false, false, 0)
	vbox.Add(u.roster.Window)
	u.window.Add(vbox)

	u.window.SetTitle("Coy")
	u.window.Connect("destroy", gtk.MainQuit)
	u.window.SetSizeRequest(200, 600)
	u.window.ShowAll()
}

func (u *gtkUI) sendMessage(to, message string) {
	//TODO: this should not be in both GUI and roster
	conversation := u.session.getConversationWith(to)

	toSend, err := conversation.Send(otr3.ValidMessage(message))
	if err != nil {
		fmt.Println("Failed to generate OTR message")
		return
	}

	encrypted := conversation.IsEncrypted()
	u.roster.AppendMessageToHistory(to, "ME", "NOW", encrypted, ui.StripHTML([]byte(message)))

	for _, m := range toSend {
		//TODO: this should be session.Send(to, message)
		u.session.conn.Send(to, string(m))
	}
}

func (u *gtkUI) checkEncrypted(to string) bool {
	c := u.session.getConversationWith(to)
	return c.IsEncrypted()
}

func (*gtkUI) AskForPassword(*config.Config) (string, error) {
	//TODO
	return "", nil
}

func (*gtkUI) Enroll(*config.Config) bool {
	//TODO
	return false
}

func authors() []string {
	if b, err := exec.Command("git", "log").Output(); err == nil {
		lines := strings.Split(string(b), "\n")

		var a []string
		r := regexp.MustCompile(`^Author:\s*([^ <]+).*$`)
		for _, e := range lines {
			ms := r.FindStringSubmatch(e)
			if ms == nil {
				continue
			}
			a = append(a, ms[1])
		}
		sort.Strings(a)
		var p string
		lines = []string{}
		for _, e := range a {
			if p == e {
				continue
			}
			lines = append(lines, e)
			p = e
		}
		lines = append(lines, "STRIKE Team <strike-public(AT)thoughtworks.com>")
		return lines
	}
	return []string{"STRIKE Team <strike-public@thoughtworks.com>"}
}

func aboutDialog() {
	dialog := gtk.NewAboutDialog()
	dialog.SetName("Coy IM!")
	dialog.SetProgramName("Coyim")
	dialog.SetAuthors(authors())
	// dir, _ := path.Split(os.Args[0])
	// imagefile := path.Join(dir, "../../data/coyim-logo.png")
	// pixbuf, _ := gdkpixbuf.NewFromFile(imagefile)
	// dialog.SetLogo(pixbuf)
	dialog.SetLicense(`Copyright (c) 2012 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`)
	dialog.SetWrapLicense(true)
	dialog.Run()
	dialog.Destroy()
}

func accountDialog() {
	//TODO It should not load config here
	c := &config.Config{}
	dialog := gtk.NewDialog()
	dialog.SetTitle("Account Details")
	dialog.SetPosition(gtk.WIN_POS_CENTER)
	vbox := dialog.GetVBox()

	accountLabel := gtk.NewLabel("Account:")
	vbox.Add(accountLabel)

	accountInput := gtk.NewEntry()
	accountInput.SetText(c.Account)
	accountInput.SetEditable(true)
	vbox.Add(accountInput)

	button := gtk.NewButtonWithLabel("OK")
	button.Connect("clicked", func() {
		fmt.Println(accountInput.GetText())
		dialog.Destroy()
	})
	vbox.Add(button)

	dialog.ShowAll()
}

func initMenuBar(u *gtkUI) *gtk.MenuBar {
	menubar := gtk.NewMenuBar()

	//Config -> Account
	cascademenu := gtk.NewMenuItemWithMnemonic("_Preference")
	menubar.Append(cascademenu)
	submenu := gtk.NewMenu()
	cascademenu.SetSubmenu(submenu)

	menuitem := gtk.NewMenuItemWithMnemonic("_Account")
	submenu.Append(menuitem)

	accountSubMenu := gtk.NewMenu()
	menuitem.SetSubmenu(accountSubMenu)

	connectItem := gtk.NewMenuItemWithMnemonic("_Connect")
	connectItem.Connect("activate", func() {
		if err := u.connect(); err != nil {
			fmt.Println("Failed to connect")
		}
	})
	accountSubMenu.Append(connectItem)

	disconnectItem := gtk.NewMenuItemWithMnemonic("_Disconnect")
	disconnectItem.Connect("activate", u.disconnect)
	accountSubMenu.Append(disconnectItem)

	editItem := gtk.NewMenuItemWithMnemonic("_Edit")
	editItem.Connect("activate", accountDialog)
	accountSubMenu.Append(editItem)

	//Help -> About
	cascademenu = gtk.NewMenuItemWithMnemonic("_Help")
	menubar.Append(cascademenu)
	submenu = gtk.NewMenu()
	cascademenu.SetSubmenu(submenu)
	menuitem = gtk.NewMenuItemWithMnemonic("_About")
	menuitem.Connect("activate", aboutDialog)
	submenu.Append(menuitem)
	return menubar
}

func (u *gtkUI) ProcessPresence(stanza *xmpp.ClientPresence, ignore, gone bool) {

	jid := xmpp.RemoveResourceFromJid(stanza.From)
	state, ok := u.session.knownStates[jid]
	if !ok || len(state) == 0 {
		state = "unknown"
	}

	//TODO: Notify via UI
	fmt.Println(jid, "is", state)
}

func (u *gtkUI) IQReceived(string) {
	//TODO
}

//TODO: we should update periodically (like Pidgin does) if we include the status (online/offline/away) on the label
func (u *gtkUI) RosterReceived(roster []xmpp.RosterEntry) {
	glib.IdleAdd(func() bool {
		u.roster.Update(roster)
		return false
	})
}

func main() {
	flag.Parse()

	ui := NewGTK()

	//ticker := time.NewTicker(1 * time.Second)
	//quit := make(chan bool)
	//go timeoutLoop(&s, ticker.C)

	ui.Loop()
	os.Stdout.Write([]byte("\n"))
}

func (u *gtkUI) disconnect() error {
	if !u.connected {
		return nil
	}

	u.session.Terminate()
	u.connected = false

	return nil
}

func (u *gtkUI) connect() error {
	if u.connected {
		return nil
	}

	var password string
	var err error

	u.config, password, err = loadConfig(u)
	if err != nil {
		return err
	}

	//TODO support one session per account
	u.session = &Session{
		ui: u,

		//Why both?
		account: u.config.Account,
		config:  u.config,

		conversations:     make(map[string]*otr3.Conversation),
		eh:                make(map[string]*eventHandler),
		knownStates:       make(map[string]string),
		privateKey:        new(otr3.PrivateKey),
		pendingRosterChan: make(chan *ui.RosterEdit),
		pendingSubscribes: make(map[string]string),
		lastActionTime:    time.Now(),
		sessionHandler:    u,
	}

	u.session.privateKey.Parse(u.config.PrivateKey)
	//TODO: This should happen regardless of connecting
	fmt.Printf("Your fingerprint is %x\n", u.session.privateKey.DefaultFingerprint())

	// TODO: GTK main loop freezes unless this is run on a Go routine
	// and I have no idea why
	go func() {
		err := u.session.Connect(password)
		if err != nil {
			return
		}

		u.connected = true
		u.onConnect()
	}()

	return nil
}

func (ui *gtkUI) onConnect() {
	go ui.session.WatchTimeout()
	go ui.session.WatchRosterEvents()
	go ui.session.WatchStanzas()
}