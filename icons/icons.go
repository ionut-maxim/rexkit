// Package icons is the glyph set terminal apps draw with: Nerd Font icons,
// which the Rex apps render, and plain Unicode fallbacks for terminals
// without a Nerd Font.
package icons

import "strings"

// Set is every glyph an app draws.
type Set struct {
	Nerd bool

	Brand, Team, Project, Inbox, Issues, Mine, Repo, Tooling, Work string

	// Workflow states, in the style of Linear.
	Triage, Backlog, Todo, Started, Review, Done, Canceled string

	Urgent string // priority bars are drawn, not glyphs

	User, Label, Branch, Comment, Calendar, Clock, Overdue string
	Check, Cross, Spinner, Info, Warning, Dot, Edit        string
	DoIt, Decide, Arrow, Chevron, Reply, Link, Image       string

	Filter, Back, BoxOn, BoxOff string
	Bullets                     []string
	Globe, App                  string
	Copy, Mail, Plus, Search    string
	Power                       string

	PillL, PillR, Sep string
}

// Nerd is the Nerd Font set.
var Nerd = Set{
	Nerd:  true,
	Brand: "", Team: "", Project: "", Inbox: "", Issues: "", Mine: "",
	Repo: "", Tooling: "", Work: "",
	Triage: "", Backlog: "", Todo: "", Started: "", Review: "", Done: "", Canceled: "",
	Urgent: "",
	User:   "", Label: "", Branch: "", Comment: "", Calendar: "", Clock: "", Overdue: "",
	Check: "", Cross: "", Spinner: "", Info: "", Warning: "", Dot: "", Edit: "",
	DoIt: "", Decide: "", Arrow: "", Chevron: "", Reply: "", Link: "", Image: "",
	Filter: "\uf0b0", Back: "\uf053", BoxOn: "\uf14a", BoxOff: "\uf096", Bullets: []string{"•", "◦", "▪"}, Globe: "\uf0ac", App: "\uf10b",
	Copy: "\uf0c5", Mail: "\uf0e0", Plus: "\uf067", Search: "\uf002", Power: "\uf011",
	PillL: "", PillR: "", Sep: "",
}

// Unicode is the plain Unicode set.
var Unicode = Set{
	Brand: "◆", Team: "◇", Project: "▣", Inbox: "✉", Issues: "≡", Mine: "☺", Repo: "⑂", Tooling: "⚙", Work: "▶",
	Triage: "?", Backlog: "◌", Todo: "○", Started: "◐", Review: "◑", Done: "●", Canceled: "⊘",
	Urgent: "!",
	User:   "☺", Label: "#", Branch: "⑂", Comment: "…", Calendar: "◷", Clock: "◷", Overdue: "!",
	Check: "✓", Cross: "✗", Spinner: "⟳", Info: "i", Warning: "!", Dot: "●", Edit: "✎",
	DoIt: "▸", Decide: "?", Arrow: "→", Chevron: "›", Reply: "↳", Link: "↗", Image: "▧",
	Filter: "▾", Back: "‹", BoxOn: "☑", BoxOff: "☐", Bullets: []string{"•", "◦", "▪"}, Globe: "↗", App: "▸",
	Copy: "⧉", Mail: "✉", Plus: "+", Search: "⌕", Power: "◉",
	PillL: "", PillR: "", Sep: "›",
}

// Current is the set in use; an app picks it at start (see Pick).
var Current = Nerd

// Pick chooses a set by name: "unicode", "plain", "off" or "no" give
// Unicode, anything else Nerd.
func Pick(name string) Set {
	switch strings.ToLower(name) {
	case "unicode", "plain", "off", "no":
		return Unicode
	}
	return Nerd
}
