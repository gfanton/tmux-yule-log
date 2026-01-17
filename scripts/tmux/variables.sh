#!/usr/bin/env bash
#
# Tmux option variables for yule-log plugin
#

# Default values
readonly default_idle_time="300"           # 5 minutes
readonly default_mode="fire"               # "fire" or "contribs"
readonly default_show_ticker="on"          # "on" or "off"
readonly default_lock_enabled="off"        # "on" or "off"
readonly default_lock_timeout="0"          # 0 = manual only
readonly default_lock_socket_protect="on"  # "on" or "off"

# Minimum supported tmux version
readonly supported_tmux_version="3.2"
