package app

import "slices"

var TwitchScopes = []string{
	"user:read:chat",
	"user:write:chat",
	"user:bot",
	"channel:bot",
	"bits:read",
  "channel:manage:broadcast",
  "channel:read:hype_train",
  "channel:manage:polls",
  "channel:manage:predictions",
  "channel:manage:redemptions",
  "channel:read:subscriptions",
  "channel:read:vips",
  "channel:moderate",
  "moderator:manage:announcements",
  "moderator:manage:banned_users",
  "moderator:manage:blocked_terms",
  "moderator:manage:chat_settings",
  "moderator:manage:unban_requests",
  "moderator:manage:chat_messages",
  "moderator:manage:warnings",
  "moderator:read:moderators",
  "moderator:read:vips",
}

func IsScopesEqual(first []string, second []string) bool {
  if len(first) != len(second) {
    return false
  }

  for _, elem := range first {
    exists := slices.Contains(second, elem)
    if !exists {
      return false
    }
  }

  return true
}
