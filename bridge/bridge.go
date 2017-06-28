package bridge

import (
	"fmt"

	"github.com/pkg/errors"
)

// Options to be passed to New
type Options struct {
	DiscordBotToken, GuildID string

	ChannelMappings map[string]string

	IRCServer       string
	IRCUseTLS       bool
	IRCListenerName string // i.e, "DiscordBot", required to listen for messages in all cases
	WebIRCPass      string
}

// A Bridge represents a bridging between an IRC server and channels in a Discord server
type Bridge struct {
	ircServerAddress string
	ircPrimaryName   string

	chanMapToIRC     map[string]string
	chanMapToDiscord map[string]string
	chanIRC          []string
	chanDiscord      []string

	h *home
}

// Close the Bridge
func (b *Bridge) Close() {
	b.h.done <- true
	<-b.h.done
}

// TODO: Use errors package
func (b *Bridge) load(opts Options) bool {
	if opts.IRCServer == "" {
		fmt.Println("Missing server name.")
		return false
	}

	b.ircServerAddress = opts.IRCServer
	b.ircPrimaryName = opts.IRCListenerName

	b.chanMapToIRC = opts.ChannelMappings

	ircChannels := make([]string, len(b.chanMapToIRC))
	discordChannels := make([]string, len(b.chanMapToIRC))

	i := 0
	for discord, irc := range opts.ChannelMappings {
		ircChannels[i] = irc
		discordChannels[i] = discord
		i++
	}

	chanMapToDiscord := make(map[string]string)
	for k, v := range b.chanMapToIRC {
		chanMapToDiscord[v] = k
	}
	b.chanMapToDiscord = chanMapToDiscord

	b.chanIRC = ircChannels
	b.chanDiscord = discordChannels

	return true
}

// New Bridge
func New(opts Options) (*Bridge, error) {
	dib := &Bridge{}
	if !dib.load(opts) {
		return nil, errors.New("error with Options. TODO: More info here")
	}

	discord, err := prepareDiscord(dib, opts.DiscordBotToken, opts.GuildID)
	ircPrimary := prepareIRCListener(dib, opts.WebIRCPass)
	ircManager := prepareIRCManager(opts.IRCServer, opts.WebIRCPass)

	if err != nil {
		return nil, err
	}

	prepareHome(dib, discord, ircPrimary, ircManager)

	discord.h = dib.h
	ircPrimary.h = dib.h
	ircManager.h = dib.h

	return dib, nil
}

// Open all the connections required to run the bridge
func (b *Bridge) Open() (err error) {

	// Open a websocket connection to Discord and begin listening.
	err = b.h.discord.Open()
	if err != nil {
		return errors.Wrap(err, "can't open discord")
	}

	err = b.h.ircListener.Connect(b.ircServerAddress)
	if err != nil {
		return errors.Wrap(err, "can't open irc connection")
	}

	go b.h.ircListener.Loop()

	return
}

// func testingChannels(id string) bool {
// 	// inf1, bottest
// 	return /*(id == "315278744572919809") ||*/ (id == "316038111811600387")
// }
