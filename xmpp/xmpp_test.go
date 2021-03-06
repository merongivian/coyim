package xmpp

import (
	"encoding/xml"
	"io"
	"reflect"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type XmppSuite struct{}

var _ = Suite(&XmppSuite{})

func (s *XmppSuite) TestDiscoReplyVerSimple(c *C) {
	expect := "QgayPKawpkPSDYmwT/WM94uAlu0="
	input := []byte(`
  <query xmlns='http://jabber.org/protocol/disco#info'
         node='http://code.google.com/p/exodus#QgayPKawpkPSDYmwT/WM94uAlu0='>
    <identity category='client' name='Exodus 0.9.1' type='pc'/>
    <feature var='http://jabber.org/protocol/caps'/>
    <feature var='http://jabber.org/protocol/disco#info'/>
    <feature var='http://jabber.org/protocol/disco#items'/>
    <feature var='http://jabber.org/protocol/muc'/>
  </query>
  `)
	var dr DiscoveryReply
	c.Assert(xml.Unmarshal(input, &dr), IsNil)
	hash, err := dr.VerificationString()
	c.Assert(err, IsNil)
	c.Assert(hash, Equals, expect)
}

func (s *XmppSuite) TestDiscoReplyVerComplex(c *C) {
	expect := "q07IKJEyjvHSyhy//CH0CxmKi8w="
	input := []byte(`
  <query xmlns='http://jabber.org/protocol/disco#info'
         node='http://psi-im.org#q07IKJEyjvHSyhy//CH0CxmKi8w='>
    <identity xml:lang='en' category='client' name='Psi 0.11' type='pc'/>
    <identity xml:lang='el' category='client' name='Ψ 0.11' type='pc'/>
    <feature var='http://jabber.org/protocol/caps'/>
    <feature var='http://jabber.org/protocol/disco#info'/>
    <feature var='http://jabber.org/protocol/disco#items'/>
    <feature var='http://jabber.org/protocol/muc'/>
    <x xmlns='jabber:x:data' type='result'>
      <field var='FORM_TYPE' type='hidden'>
        <value>urn:xmpp:dataforms:softwareinfo</value>
      </field>
      <field var='ip_version'>
        <value>ipv4</value>
        <value>ipv6</value>
      </field>
      <field var='os'>
        <value>Mac</value>
      </field>
      <field var='os_version'>
        <value>10.5.1</value>
      </field>
      <field var='software'>
        <value>Psi</value>
      </field>
      <field var='software_version'>
        <value>0.11</value>
      </field>
    </x>
  </query>
`)
	var dr DiscoveryReply
	c.Assert(xml.Unmarshal(input, &dr), IsNil)
	hash, err := dr.VerificationString()
	c.Assert(err, IsNil)
	c.Assert(hash, Equals, expect)
}

func (s *XmppSuite) TestConnClose(c *C) {
	mockIn := &mockConnIOReaderWriter{
		read: []byte("<?xml version='1.0'?><str:stream xmlns:str='http://etherx.jabber.org/streams' version='1.0'></str:stream>"),
	}
	mockCloser := &mockConnIOReaderWriter{}
	conn := NewConn(xml.NewDecoder(mockIn), mockCloser, "")

	// consumes the opening stream
	nextElement(conn.in)
	go conn.Next()

	// blocks until it receives the </stream> or timeouts
	c.Assert(conn.Close(), IsNil)
	c.Assert(mockCloser.calledClose, Equals, 1)
	c.Assert(mockCloser.write, DeepEquals, []byte("</stream:stream>"))
}

func (s *XmppSuite) TestConnNextEOF(c *C) {
	mockIn := &mockConnIOReaderWriter{err: io.EOF}
	conn := Conn{
		in: xml.NewDecoder(mockIn),
	}
	stanza, err := conn.Next()
	c.Assert(stanza.Name, Equals, xml.Name{})
	c.Assert(stanza.Value, IsNil)
	c.Assert(err, Equals, io.EOF)
}

func (s *XmppSuite) TestConnNextErr(c *C) {
	mockIn := &mockConnIOReaderWriter{
		read: []byte(`
      <field var='os'>
        <value>Mac</value>
      </field>
		`),
	}
	conn := Conn{
		in: xml.NewDecoder(mockIn),
	}
	stanza, err := conn.Next()
	c.Assert(stanza.Name, Equals, xml.Name{})
	c.Assert(stanza.Value, IsNil)
	c.Assert(err.Error(), Equals, "unexpected XMPP message  <field/>")
}

func (s *XmppSuite) TestConnNextIQSet(c *C) {
	mockIn := &mockConnIOReaderWriter{
		read: []byte(`
<iq to='example.com'
    xmlns='jabber:client'
    type='set'
    id='sess_1'>
  <session xmlns='urn:ietf:params:xml:ns:xmpp-session'/>
</iq>
  `),
	}
	conn := Conn{
		in: xml.NewDecoder(mockIn),
	}
	stanza, err := conn.Next()
	c.Assert(stanza.Name, Equals, xml.Name{Space: NsClient, Local: "iq"})
	iq, ok := stanza.Value.(*ClientIQ)
	c.Assert(ok, Equals, true)
	c.Assert(iq.To, Equals, "example.com")
	c.Assert(iq.Type, Equals, "set")
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnNextIQResult(c *C) {
	mockIn := &mockConnIOReaderWriter{
		read: []byte(`
<iq from='example.com'
    xmlns='jabber:client'
    type='result'
    id='sess_1'/>
  `),
	}
	conn := Conn{
		in: xml.NewDecoder(mockIn),
	}
	stanza, err := conn.Next()
	c.Assert(stanza.Name, Equals, xml.Name{Space: NsClient, Local: "iq"})
	iq, ok := stanza.Value.(*ClientIQ)
	c.Assert(ok, Equals, true)
	c.Assert(iq.From, Equals, "example.com")
	c.Assert(iq.Type, Equals, "result")
	c.Assert(err, ErrorMatches, "xmpp: failed to parse id from iq: .*")
}

func (s *XmppSuite) TestConnCancelError(c *C) {
	conn := Conn{}
	ok := conn.Cancel(conn.getCookie())
	c.Assert(ok, Equals, false)
}

func (s *XmppSuite) TestConnCancelOK(c *C) {
	conn := Conn{}
	cookie := conn.getCookie()
	ch := make(chan Stanza, 1)
	conn.inflights = make(map[Cookie]inflight)
	conn.inflights[cookie] = inflight{ch, ""}
	ok := conn.Cancel(cookie)
	c.Assert(ok, Equals, true)
	_, ok = conn.inflights[cookie]
	c.Assert(ok, Equals, false)
}

func (s *XmppSuite) TestConnRequestRoster(c *C) {
	mockOut := mockConnIOReaderWriter{}
	conn := Conn{
		out: &mockOut,
	}
	conn.inflights = make(map[Cookie]inflight)
	ch, cookie, err := conn.RequestRoster()
	c.Assert(string(mockOut.write), Matches, "<iq type='get' id='.*'><query xmlns='jabber:iq:roster'/></iq>")
	c.Assert(ch, NotNil)
	c.Assert(cookie, NotNil)
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnRequestRosterErr(c *C) {
	mockOut := mockConnIOReaderWriter{err: io.EOF}
	conn := Conn{
		out: &mockOut,
	}
	conn.inflights = make(map[Cookie]inflight)
	ch, cookie, err := conn.RequestRoster()
	c.Assert(string(mockOut.write), Matches, "<iq type='get' id='.*'><query xmlns='jabber:iq:roster'/></iq>")
	c.Assert(ch, IsNil)
	c.Assert(cookie, NotNil)
	c.Assert(err, Equals, io.EOF)
}

func (s *XmppSuite) TestParseRoster(c *C) {
	iq := ClientIQ{}
	iq.Query = []byte(`
  <query xmlns='jabber:iq:roster'>
    <item jid='romeo@example.net'
          name='Romeo'
          subscription='both'>
      <group>Friends</group>
    </item>
    <item jid='mercutio@example.org'
          name='Mercutio'
          subscription='from'>
      <group>Friends</group>
    </item>
    <item jid='benvolio@example.org'
          name='Benvolio'
          subscription='both'>
      <group>Friends</group>
    </item>
  </query>
  `)
	reply := Stanza{
		Value: &iq,
	}
	rosterEntrys, err := ParseRoster(reply)
	c.Assert(rosterEntrys, NotNil)
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnSendIQReplyAndTyp(c *C) {
	mockOut := mockConnIOReaderWriter{}
	conn := Conn{
		out: &mockOut,
		jid: "jid",
	}
	conn.inflights = make(map[Cookie]inflight)
	reply, cookie, err := conn.SendIQ("example@xmpp.com", "typ", nil)
	c.Assert(string(mockOut.write), Matches, "<iq to='example@xmpp.com' from='jid' type='typ' id='.*'></iq>")
	c.Assert(reply, NotNil)
	c.Assert(cookie, NotNil)
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnSendIQErr(c *C) {
	mockOut := mockConnIOReaderWriter{err: io.EOF}
	conn := Conn{
		out: &mockOut,
		jid: "jid",
	}
	reply, cookie, err := conn.SendIQ("example@xmpp.com", "typ", nil)
	c.Assert(string(mockOut.write), Matches, "<iq to='example@xmpp.com' from='jid' type='typ' id='.*'>$")
	c.Assert(reply, NotNil)
	c.Assert(cookie, NotNil)
	c.Assert(err, Equals, io.EOF)
}

func (s *XmppSuite) TestConnSendIQEmptyReply(c *C) {
	mockOut := mockConnIOReaderWriter{}
	conn := Conn{
		out: &mockOut,
		jid: "jid",
	}
	conn.inflights = make(map[Cookie]inflight)
	reply, cookie, err := conn.SendIQ("example@xmpp.com", "typ", reflect.ValueOf(EmptyReply{}))
	c.Assert(string(mockOut.write), Matches, "<iq to='example@xmpp.com' from='jid' type='typ' id='.*'><Value><flag>.*</flag></Value></iq>")
	c.Assert(reply, NotNil)
	c.Assert(cookie, NotNil)
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnSendIQReply(c *C) {
	mockOut := mockConnIOReaderWriter{}
	conn := Conn{
		out: &mockOut,
		jid: "jid",
	}
	err := conn.SendIQReply("example@xmpp.com", "typ", "id", nil)
	c.Assert(string(mockOut.write), Matches, "<iq to='example@xmpp.com' from='jid' type='typ' id='id'></iq>")
	c.Assert(err, IsNil)
}

func (s *XmppSuite) TestConnSend(c *C) {
	mockOut := mockConnIOReaderWriter{}
	conn := Conn{
		out: &mockOut,
		jid: "jid",
	}
	err := conn.Send("example@xmpp.com", "message")
	c.Assert(string(mockOut.write), Matches, "<message to='example@xmpp.com' from='jid' type='chat'><body>message</body><nos:x xmlns:nos='google:nosave' value='enabled'/><arc:record xmlns:arc='http://jabber.org/protocol/archive' otr='require'/></message>")
	c.Assert(err, IsNil)
}
