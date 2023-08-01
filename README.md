# TIRED
## Why?
**Tired** stands for **TIme REcorDer** and should help anybody to log his work to Jira issues. For me is much more convenient to log my work at CSV text file using vim than to use Jira UI.
So using tired one could write any number of formated lines to a file using special format and then just run the utility to push records via Jira API. Only basic auth is supported right now but it's not a problem to add any other type supported by [andygrunwald/go-jira](https://github.com/andygrunwald/go-jira).

## Timesheet file format
Colons are: date, start time, end time, Jira issue, commentary.

Example:
```
# Lines starting with '#' and empty lines are ignored

2023-07-31,16:00,17:00,SRE-899,"creating a fancy script"
2023-07-31,17:00,19:00,SRE-838,"observing modern GNU/Linux distributions"

2023-08-01,10:10,10:40,SRE-187,"read email etc"
2023-08-01,10:40,12:10,SRE-838,"more reading about distributions"
```

After reporting all lines **Tired** moves timesheet file to the same path but with `.bak` suffix and creates a new timesheet file with the only change. It adds or moves `>>> TIRED <<<` line, which I call marker, to the end of file. During next run **tired** will only push lines below marker. If one wants to re-send any lines, he can move them below `>>> TIRED <<<` marker and restart the utility.
