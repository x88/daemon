package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type ubuntuSysVRecord struct {
	name        string
	description string
}

// Standard service path for systemV daemons
func (linux *ubuntuSysVRecord) servicePath() string {
	return "/etc/init.d/" + linux.name
}

// Check service is installed
func (linux *ubuntuSysVRecord) checkInstalled() bool {

	if _, err := os.Stat(linux.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (linux *ubuntuSysVRecord) checkRunning() (string, bool) {
	output, err := exec.Command("service", linux.name, "status").Output()
	if err == nil && strings.Contains(string(output), linux.name) {
		return string(output), strings.Contains(string(output), "is running")
	}
	return "Service is stopped", false
}

func (linux *ubuntuSysVRecord) InstallFromPath(thePath string) (string, error) {
	installAction := "Install " + linux.description + ":"

	if checkPrivileges() == false {
		return installAction + failed, errors.New(rootPrivileges)
	}

	srvPath := linux.servicePath()

	if linux.checkInstalled() == true {
		return installAction + failed, errors.New(linux.description + " already installed")
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	isexec, err := IsExecutable(thePath)
	if err != nil {
		return installAction + failed, err
	}
	if !isexec {
		return installAction + failed, fmt.Errorf("target is not executable: %s", thePath)
	}

	templ, err := template.New("ubuntuSysVConfig").Parse(ubuntuSysVConf)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Path string
		}{linux.name, linux.description, thePath},
	); err != nil {
		return installAction + failed, err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return installAction + failed, err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/S87"+linux.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/K17"+linux.name); err != nil {
			continue
		}
	}

	return installAction + success, nil
}

// Install the service
func (linux *ubuntuSysVRecord) Install() (string, error) {
	installAction := "Install " + linux.description + ":"

	execPatch, err := executablePath(linux.name)
	if err != nil {
		return installAction + failed, err
	}
	return linux.InstallFromPath(execPatch)
}

// Remove the service
func (linux *ubuntuSysVRecord) Remove() (string, error) {
	removeAction := "Removing " + linux.description + ":"

	if checkPrivileges() == false {
		return removeAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return removeAction + failed, errors.New(linux.description + " is not installed")
	}

	if err := os.Remove(linux.servicePath()); err != nil {
		return removeAction + failed, err
	}

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Remove("/etc/rc" + i + ".d/S87" + linux.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Remove("/etc/rc" + i + ".d/K17" + linux.name); err != nil {
			continue
		}
	}

	return removeAction + success, nil
}

// Start the service
func (linux *ubuntuSysVRecord) Start() (string, error) {
	startAction := "Starting " + linux.description + ":"

	if checkPrivileges() == false {
		return startAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return startAction + failed, errors.New(linux.description + " is not installed")
	}

	if _, status := linux.checkRunning(); status == true {
		return startAction + failed, errors.New("service already running")
	}

	if err := exec.Command("service", linux.name, "start").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (linux *ubuntuSysVRecord) Stop() (string, error) {
	stopAction := "Stopping " + linux.description + ":"

	if checkPrivileges() == false {
		return stopAction + failed, errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return stopAction + failed, errors.New(linux.description + " is not installed")
	}

	if _, status := linux.checkRunning(); status == false {
		return stopAction + failed, errors.New("service already stopped")
	}

	if err := exec.Command("service", linux.name, "stop").Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (linux *ubuntuSysVRecord) Status() (string, error) {

	if checkPrivileges() == false {
		return "", errors.New(rootPrivileges)
	}

	if linux.checkInstalled() == false {
		return "Status could not defined", errors.New(linux.description + " is not installed")
	}

	statusAction, _ := linux.checkRunning()

	return statusAction, nil
}

var ubuntuSysVConf = `
#! /bin/sh
### BEGIN INIT INFO
# Provides:          {{.Name}}
# Required-Start:    $network $named
# Required-Stop:     $network $named
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.Description}}
# Description:       {{.Description}}
### END INIT INFO

# Do NOT "set -e"

# PATH should only include /usr/* if it runs after the mountnfs.sh script
PATH=/sbin:/usr/sbin:/bin:/usr/bin
DESC="{{.Description}}"
NAME={{.Name}} 
DAEMON="{{.Path}}"
DAEMON_ARGS=""
PIDFILE=/var/run/$NAME.pid
LOGFILE=/var/log/$NAME.log
SCRIPTNAME=/etc/init.d/$NAME

# Exit if the package is not installed
[ -x "$DAEMON" ] || exit 0

# Read configuration variable file if it is present
[ -r /etc/default/$NAME ] && . /etc/default/$NAME

# Load the VERBOSE setting and other rcS variables
. /lib/init/vars.sh

# Define LSB log_* functions.
# Depend on lsb-base (>= 3.2-14) to ensure that this file is present
# and status_of_proc is working.
. /lib/lsb/init-functions

#
# Function that starts the daemon/service
#
do_start()
{
    # Return
    #   0 if daemon has been started
    #   1 if daemon was already running
    #   2 if daemon could not be started
    start-stop-daemon --start --background --make-pidfile --quiet --pidfile $PIDFILE --exec $DAEMON --test > /dev/null \
        || return 1
    start-stop-daemon --start --background --make-pidfile --quiet --pidfile $PIDFILE --exec $DAEMON -- \
        $DAEMON_ARGS \
        || return 2
    # Add code here, if necessary, that waits for the process to be ready
    # to handle requests from services started subsequently which depend
    # on this one.  As a last resort, sleep for some time.
}

#
# Function that stops the daemon/service
#
do_stop()
{
    # Return
    #   0 if daemon has been stopped
    #   1 if daemon was already stopped
    #   2 if daemon could not be stopped
    #   other if a failure occurred
    start-stop-daemon --stop --quiet --retry 0 --pidfile $PIDFILE --name $NAME
    RETVAL="$?"
    [ "$RETVAL" = 2 ] && return 2
    # Wait for children to finish too if this is a daemon that forks
    # and if the daemon is only ever run from this initscript.
    # If the above conditions are not satisfied then add some other code
    # that waits for the process to drop all resources that could be
    # needed by services started subsequently.  A last resort is to
    # sleep for some time.
    start-stop-daemon --stop --quiet --oknodo --retry 0 --exec $DAEMON
    [ "$?" = 2 ] && return 2
    # Many daemons don't delete their pidfiles when they exit.
    rm -f $PIDFILE
    return "$RETVAL"
}

#
# Function that sends a SIGHUP to the daemon/service
#
do_reload() {
    #
    # If the daemon can reload its configuration without
    # restarting (for example, when it is sent a SIGHUP),
    # then implement that here.
    #
    start-stop-daemon --stop --signal 1 --quiet --pidfile $PIDFILE --name $NAME
    return 0
}

case "$1" in
  start)
    [ "$VERBOSE" != no ] && log_daemon_msg "Starting $DESC" "$NAME"
    do_start
    case "$?" in
        0|1) [ "$VERBOSE" != no ] && log_end_msg 0 ;;
        2) [ "$VERBOSE" != no ] && log_end_msg 1 ;;
    esac
    ;;
  stop)
    [ "$VERBOSE" != no ] && log_daemon_msg "Stopping $DESC" "$NAME"
    do_stop
    case "$?" in
        0|1) [ "$VERBOSE" != no ] && log_end_msg 0 ;;
        2) [ "$VERBOSE" != no ] && log_end_msg 1 ;;
    esac
    ;;
  status)
    status_of_proc "$DAEMON" "$NAME" && exit 0 || exit $?
    ;;
  #reload|force-reload)
    #
    # If do_reload() is not implemented then leave this commented out
    # and leave 'force-reload' as an alias for 'restart'.
    #
    #log_daemon_msg "Reloading $DESC" "$NAME"
    #do_reload
    #log_end_msg $?
    #;;
  restart|force-reload)
    #
    # If the "reload" option is implemented then remove the
    # 'force-reload' alias
    #
    log_daemon_msg "Restarting $DESC" "$NAME"
    do_stop
    case "$?" in
      0|1)
        do_start
        case "$?" in
            0) log_end_msg 0 ;;
            1) log_end_msg 1 ;; # Old process is still running
            *) log_end_msg 1 ;; # Failed to start
        esac
        ;;
      *)
        # Failed to stop
        log_end_msg 1
        ;;
    esac
    ;;
  *)
    #echo "Usage: $SCRIPTNAME {start|stop|restart|reload|force-reload}" >&2
    echo "Usage: $SCRIPTNAME {start|stop|status|restart|force-reload}" >&2
    exit 3
    ;;
esac

:

`
