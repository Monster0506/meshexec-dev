package tui

// Simple text/emoji icons; callers can choose to disable emoji via flags.
type Icons struct {
	Check string
	Cross string
	Info  string
	Warn  string
	Spark string
	Peer  string
	Run   string
}

func defaultIcons(useEmoji bool) Icons {
	if useEmoji {
		return Icons{
			Check: "✅",
			Cross: "❌",
			Info:  "💡",
			Warn:  "⚠️",
			Spark: "✨",
			Peer:  "🟢",
			Run:   "🚀",
		}
	}
	return Icons{
		Check: "[OK]",
		Cross: "[X]",
		Info:  "[i]",
		Warn:  "[!]",
		Spark: "*",
		Peer:  "●",
		Run:   ">",
	}
}
