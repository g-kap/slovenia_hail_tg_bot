package tgbot

type Cmd string

const (
	CmdHelp              Cmd = "/help"
	CmdListSubscriptions Cmd = "/list"
	CmdUnsubscribeAll    Cmd = "/unsubscribe_all"
	CmdSubscribe         Cmd = "/subscribe"
)

func allCommands() []Cmd {
	return []Cmd{
		CmdHelp, CmdListSubscriptions, CmdUnsubscribeAll, CmdSubscribe,
	}
}

func (cmd Cmd) valid() bool {
	for _, c := range allCommands() {
		if cmd == c {
			return true
		}
	}
	return false
}

func (cmd Cmd) desc() string {
	switch cmd {
	case CmdHelp:
		return "print this help"
	case CmdListSubscriptions:
		return "print all subscriptions"
	case CmdUnsubscribeAll:
		return "unsubscribe from all regions"
	case CmdSubscribe:
		return "subscribe on specific region"
	}
	return ""
}

func makeHelpText() string {
	txt := ""
	for _, c := range allCommands() {
		txt += string(c) + " " + c.desc() + "\n"
	}
	return txt
}
