# 0.14.0 / 2025-09-20

  * refactor: move modules into own packages, cleanup code, modernize code
  * fix(deps): update module github.com/bwmarrin/discordgo to v0.29.0
  * fix(deps): update module github.com/luzifer/korvike/functions to v1.0.2
  * fix(deps): update module github.com/luzifer/rconfig/v2 to v2.6.0
  * chore(deps): update dependency go to v1.25.1
  * chore: replace gopkg.in/yaml.v2

# 0.13.0 / 2024-12-12

  * [liveposting] Make embed title and therefore whole embed optional
  * [streamSchedule] Add capability for content templating in stream schedule
  * Fix: Only access slice if it has content
  * Fix: Expose `formatTime` formatter instead for timezon support
  * Fix: Allow locale time in templating
  * Fix: Compare content for updates
  * Fix: Do not try to compare non existing embed
  * Fix: Remove `embed_title` as required attribute
  * Update Go dependencies

# 0.12.3 / 2022-03-04

  * Fix: Trim spaces in title as Discord does

# 0.12.2 / 2022-03-03

  * Update dependencies

# 0.12.1 / 2022-02-11

  * Lint: Disable gocyclo triggered by switch statement

# 0.12.0 / 2022-02-11

  * [streamSchedule] Improve display for empty stream title

# 0.11.1 / 2022-02-11

  * Update dependencies

# 0.11.0 / 2021-12-07

  * [streamSchedule] Fix: To not append game name if title contains it

# 0.10.0 / 2021-11-16

  * [liveposting] Add option to remove older messages for the same channel
  * [liveposting] Allow to override the post text for specific users

# 0.9.0 / 2021-11-05

  * [core] Update dependencies, switch to go1.17 go.mod format
  * [liveposting] Fix: Work around Discord screwing up image URLs

# 0.8.0 / 2021-10-26

  * [liveposting] Add support for preserve-proxy for stream previews
  * [liveposting] Add some debug logging to post creation
  * [liverole] Fix: Take username from member information
  * [liverole] Log username for add / remove decisions

# 0.7.0 / 2021-08-28

  * [livePosting] Increase preview size
  * Wiki: Improve examples with code tags
  * [streamSchedule] Support dynamic date specification
  * Wiki: Add more documentation for config parameters
  * Add info about the guild\_id
  * Add check for given guild ID

# 0.6.1 / 2021-08-26

  * Fix: Add reason, move role change to debug level

# 0.6.0 / 2021-08-26

  * Fix: Log version of bot currently started
  * Add logging for successful role change
  * Fix: Prevent activating duplicate module ID
  * Wiki: Document role hierarchy
  * Fix: Lock live-posts to prevent double posting

# 0.5.0 / 2021-08-23

  * Switch presence module to app-access-token
  * Fix: Log setup finish to give user a clue the bot is ready
  * Fix: Handle empty store-file created for permission management

# 0.4.0 / 2021-08-21

  * Sec: Use app-access-token instead of user-token for schedule fetching
  * Add reactionrole module. handle optional thumbnail in streamschedule better
  * Instead of scanning for message, use store to save its ID
  * [module/liveposting] Add auto\_publish functionality
  * Add persistent store and module IDs
  * Add setup method to execute actions after connect

# 0.3.1 / 2021-08-06

  * Fix: Do not break posting on one non-fresh stream
  * Fix/Optimize: Prevent duplicate attribute parsing

# 0.3.0 / 2021-08-06

  * Lint: Handle URL parser error for stream previews
  * Add filter for configured guild
  * Force Discord to fetch fresh previews for the streams
  * Simplify presence module and make better readable

# 0.2.0 / 2021-08-01

  * Add polling for live-streams (#1)
  * Add more documentation

# 0.1.0 / 2021-07-31

  * Initial release
