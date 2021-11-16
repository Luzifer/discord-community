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
