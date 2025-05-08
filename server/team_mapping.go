// team_mapping.go
// Mapping from teams to players
package server

import (
    "slices"
)

type Team struct {
    Name      string
    Usernames []string
} // <-- struct Team

// Player team mappings
// TODO: Just use `map`?
type TeamMapping struct {
    Data []Team
} // <-- struct TeamMapping

// Get players by team
func (self *TeamMapping) PlayerTeams(username string) []string {
    var ret []string

    for _, team := range self.Data {
        if slices.Contains(team.Usernames, username) {
            ret = append(ret, team.Name)
        }
    }

    return ret
} // <-- ([]TeamMapping)::PlayerTeams(username)


// Get teams for a player
func (self *TeamMapping) TeamPlayers(team_name string) []string {
    var ret []string

    for _, team := range self.Data {
        if team.Name == team_name {
            ret = append(ret, team.Usernames...)
        }
    }

    slices.Sort(ret)
    ret = slices.Compact(ret)

    return ret
} // <-- TeamMapping::TeamPlayers(team)
