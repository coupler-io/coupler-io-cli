package dataflows

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		name        string
		enabled     bool
		interval    string
		weekDays    []int
		monthDay    int
		at          string
		timezone    string
		activeStart int
		activeEnd   int
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Rename a dataflow or change its schedule",
		Long: `Rename a dataflow and/or change its refresh schedule.

Schedule fields by interval:
  daily, monthly           --at HH:MM
  hourly/semiHourly/quarterHourly   --active-hours-start / --active-hours-end (0-23)
  daily/hourly/semiHourly/quarterHourly   --week-day 0 (Monday) .. 6 (Sunday), repeatable
  monthly                  --month-day 1..28

--schedule-enabled=false pauses the schedule; --schedule-enabled=true resumes the previous one.`,
		Example: `  coupler-io dataflows update <id> --name "Weekly sales"
  coupler-io dataflows update <id> --interval daily --at 08:00 --timezone Europe/Berlin
  coupler-io dataflows update <id> --interval hourly --week-day 0 --week-day 1 --active-hours-start 9 --active-hours-end 18
  coupler-io dataflows update <id> --schedule-enabled=false`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			body := map[string]interface{}{}
			schedule := map[string]interface{}{}

			if flags.Changed("name") {
				body["name"] = name
			}
			if flags.Changed("schedule-enabled") {
				schedule["enabled"] = enabled
			}
			if flags.Changed("interval") {
				schedule["interval"] = interval
			}
			if flags.Changed("week-day") {
				schedule["week_days"] = weekDays
			}
			if flags.Changed("month-day") {
				schedule["month_day"] = monthDay
			}
			if flags.Changed("at") {
				schedule["time"] = at
			}
			if flags.Changed("timezone") {
				schedule["timezone"] = timezone
			}
			if flags.Changed("active-hours-start") || flags.Changed("active-hours-end") {
				if !flags.Changed("active-hours-start") || !flags.Changed("active-hours-end") {
					return cmdutil.UsageErrorf("--active-hours-start and --active-hours-end must be set together")
				}
				schedule["active_hours"] = map[string]int{"start": activeStart, "end": activeEnd}
			}
			if len(schedule) > 0 {
				body["schedule_config"] = schedule
			}
			if len(body) == 0 {
				return cmdutil.UsageErrorf("nothing to update: set --name or a schedule flag (see --help)")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", args[0])
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Patch(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Dataflow %s updated.", args[0]), nil)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "New dataflow name")
	flags.BoolVar(&enabled, "schedule-enabled", false, "Turn the schedule on or off")
	flags.StringVar(&interval, "interval", "", "Schedule interval: quarterHourly, semiHourly, hourly, daily, monthly")
	flags.IntSliceVar(&weekDays, "week-day", nil, "Day to run, 0 (Monday) to 6 (Sunday); repeatable")
	flags.IntVar(&monthDay, "month-day", 0, "Day of month to run, 1-28 (monthly)")
	flags.StringVar(&at, "at", "", "Start time as 24h HH:MM (daily, monthly)")
	flags.StringVar(&timezone, "timezone", "", "IANA timezone, e.g. America/New_York")
	flags.IntVar(&activeStart, "active-hours-start", 0, "First active hour, 0-23 (hourly intervals)")
	flags.IntVar(&activeEnd, "active-hours-end", 0, "Last active hour, 0-23 (hourly intervals)")
	return cmd
}
