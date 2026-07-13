package customisation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TicketsBot-cloud/gdl/objects"
	"github.com/TicketsBot-cloud/gdl/objects/guild/emoji"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"github.com/TicketsBot-cloud/worker/config"
)

type CustomEmoji struct {
	Name     string
	Id       uint64
	Animated bool
}

func NewCustomEmoji(name string, id uint64, animated bool) CustomEmoji {
	return CustomEmoji{
		Name:     name,
		Id:       id,
		Animated: animated,
	}
}

// Configured reports whether the bot actually has this emoji. Discord application emojis can only
// be used by the application that owns them, so a whitelabel bot renders nothing unless its owner
// uploaded emojis to its own application and stored their ids.
func (e CustomEmoji) Configured() bool {
	return e.Id != 0
}

func (e CustomEmoji) String() string {
	if e.Animated {
		return fmt.Sprintf("<a:%s:%d>", e.Name, e.Id)
	} else {
		return fmt.Sprintf("<:%s:%d>", e.Name, e.Id)
	}
}

func (e CustomEmoji) BuildEmoji() *emoji.Emoji {
	if !e.Configured() {
		return nil
	}

	return &emoji.Emoji{
		Id:       objects.NewNullableSnowflake(e.Id),
		Name:     e.Name,
		Animated: e.Animated,
	}
}

// EmojiSet is the set of emojis a single bot renders with.
type EmojiSet struct {
	Id         CustomEmoji
	Open       CustomEmoji
	OpenTime   CustomEmoji
	Close      CustomEmoji
	CloseTime  CustomEmoji
	Reason     CustomEmoji
	Subject    CustomEmoji
	Transcript CustomEmoji
	Claim      CustomEmoji
	Panel      CustomEmoji
	Rating     CustomEmoji
	Staff      CustomEmoji
	Thread     CustomEmoji
	BulletLine CustomEmoji
	Patreon    CustomEmoji
	Discord    CustomEmoji
	Logo       CustomEmoji
}

// EmojiNames are the slots a bot can configure, in the order the dashboard lists them. The name
// is also what Discord renders in <:name:id>.
var EmojiNames = []string{
	"id", "open", "opentime", "close", "closetime", "reason", "subject", "transcript",
	"claim", "panel", "rating", "staff", "thread", "bulletline", "patreon", "discord",
	"TicketsLogo",
}

func (s *EmojiSet) field(name string) *CustomEmoji {
	switch name {
	case "id":
		return &s.Id
	case "open":
		return &s.Open
	case "opentime":
		return &s.OpenTime
	case "close":
		return &s.Close
	case "closetime":
		return &s.CloseTime
	case "reason":
		return &s.Reason
	case "subject":
		return &s.Subject
	case "transcript":
		return &s.Transcript
	case "claim":
		return &s.Claim
	case "panel":
		return &s.Panel
	case "rating":
		return &s.Rating
	case "staff":
		return &s.Staff
	case "thread":
		return &s.Thread
	case "bulletline":
		return &s.BulletLine
	case "patreon":
		return &s.Patreon
	case "discord":
		return &s.Discord
	case "TicketsLogo":
		return &s.Logo
	default:
		return nil
	}
}

// IsValidEmojiName guards the dashboard against storing slots the bot never renders.
func IsValidEmojiName(name string) bool {
	var set EmojiSet
	return set.field(name) != nil
}

// DefaultEmojis is the public bot's set: owned by the public application, configured through the
// EMOJI_* environment variables.
func DefaultEmojis() EmojiSet {
	return EmojiSet{
		Id:         NewCustomEmoji("id", config.Conf.Emojis.Id, false),
		Open:       NewCustomEmoji("open", config.Conf.Emojis.Open, false),
		OpenTime:   NewCustomEmoji("opentime", config.Conf.Emojis.OpenTime, false),
		Close:      NewCustomEmoji("close", config.Conf.Emojis.Close, false),
		CloseTime:  NewCustomEmoji("closetime", config.Conf.Emojis.CloseTime, false),
		Reason:     NewCustomEmoji("reason", config.Conf.Emojis.Reason, false),
		Subject:    NewCustomEmoji("subject", config.Conf.Emojis.Subject, false),
		Transcript: NewCustomEmoji("transcript", config.Conf.Emojis.Transcript, false),
		Claim:      NewCustomEmoji("claim", config.Conf.Emojis.Claim, false),
		Panel:      NewCustomEmoji("panel", config.Conf.Emojis.Panel, false),
		Rating:     NewCustomEmoji("rating", config.Conf.Emojis.Rating, false),
		Staff:      NewCustomEmoji("staff", config.Conf.Emojis.Staff, false),
		Thread:     NewCustomEmoji("thread", config.Conf.Emojis.Thread, false),
		BulletLine: NewCustomEmoji("bulletline", config.Conf.Emojis.BulletLine, false),
		Patreon:    NewCustomEmoji("patreon", config.Conf.Emojis.Patreon, false),
		Discord:    NewCustomEmoji("discord", config.Conf.Emojis.Discord, false),
		Logo:       NewCustomEmoji("TicketsLogo", config.Conf.Emojis.Logo, false),
	}
}

type cachedEmojiSet struct {
	set     EmojiSet
	expires time.Time
}

var (
	emojiCacheMu sync.RWMutex
	emojiCache   = make(map[uint64]cachedEmojiSet)
)

const emojiCacheTtl = time.Minute

// InvalidateEmojiCache drops a bot's cached set. The dashboard shares this process with the
// worker, so saving emojis from the dashboard takes effect immediately.
func InvalidateEmojiCache(botId uint64) {
	emojiCacheMu.Lock()
	delete(emojiCache, botId)
	emojiCacheMu.Unlock()
}

// GetEmojis returns the emojis the given bot renders with. The public bot uses the env-configured
// set; a whitelabel bot uses whatever its owner stored, and renders no emoji at all for the slots
// they left empty.
func GetEmojis(ctx context.Context, botId uint64, isWhitelabel bool) EmojiSet {
	if !isWhitelabel {
		return DefaultEmojis()
	}

	emojiCacheMu.RLock()
	cached, ok := emojiCache[botId]
	emojiCacheMu.RUnlock()

	if ok && time.Now().Before(cached.expires) {
		return cached.set
	}

	stored, err := dbclient.Client.WhitelabelEmojis.GetAll(ctx, botId)
	if err != nil {
		// Degrade to "no emojis" — how whitelabel bots behaved before they could configure their
		// own — rather than risk rendering a broken <:name:0> in a user-facing message.
		return EmojiSet{}
	}

	var set EmojiSet
	for name, emoji := range stored {
		field := set.field(name)
		if field == nil {
			continue
		}

		*field = NewCustomEmoji(name, emoji.EmojiId, emoji.Animated)
	}

	emojiCacheMu.Lock()
	emojiCache[botId] = cachedEmojiSet{set: set, expires: time.Now().Add(emojiCacheTtl)}
	emojiCacheMu.Unlock()

	return set
}

// PrefixWithEmoji prefixes s with the emoji, when the bot has that emoji configured.
func PrefixWithEmoji(s string, emoji CustomEmoji) string {
	if emoji.Configured() {
		return fmt.Sprintf("%s %s", emoji, s)
	}

	return s
}
