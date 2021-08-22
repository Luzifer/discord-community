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