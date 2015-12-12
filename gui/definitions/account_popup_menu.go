package definitions

func init() {
	add(`AccountPopupMenu`, &defAccountPopupMenu{})
}

type defAccountPopupMenu struct{}

func (*defAccountPopupMenu) String() string {
	return `
<interface>
  <object class="GtkMenu" id="accountMenu">
    <child>
      <object class="GtkMenuItem" id="connectMenuItem">
        <property name="label" translatable="yes">Connect</property>
        <signal name="activate" handler="on_connect" />
      </object>
    </child>
    <child>
      <object class="GtkMenuItem" id="disconnectMenuItem">
        <property name="label" translatable="yes">Disconnect</property>
        <signal name="activate" handler="on_disconnect" />
      </object>
    </child>
    <child>
      <object class="GtkSeparatorMenuItem" id="sep1"/>
    </child>
    <child>
      <object class="GtkMenuItem" id="dumpInfoMenuItem">
        <property name="label" translatable="yes">Dump info</property>
        <signal name="activate" handler="on_dump_info" />
      </object>
    </child>
  </object>
</interface>

`
}
