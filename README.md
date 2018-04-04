### Go-RT  
#### Development of a CLI RT Tool written in GO  
Please note, this was developed/tested under RT/3.8.8  
  
### rt -h  
```
Usage: rt <subcommand> [globaloptions] [options]  
  
  Sub-commands (rt <subcommand> -h)  
    GET:    Pull Individual Ticket Data  
    UDPATE: Update Individual Ticket Data  
    CREATE: Create a Ticket  
    SEARCH: Search for tickets with given criteria  
      
  Global Options  
    -config [string]: Alt config file (~/.rt.d/config)  
    -user [string]:   Specify username for Endpoint  
    -e [string]:      Endpoint (ie: my.rtserver.com)
```  
The Global Options and more are able to be stored in a config file of your choosing. The default config file is located under `/home/username/.rt.d/config` and can be changed by using the -config flag. Inside the config file you may specify a few 'sticky' settings:  
You can throw server settings:  
```
endpoint: my.rtserver.com  
username: jsmith  
password: secret
```
as well, you can add formatting settings which come in handy using GET when you only want to see specifc information about a ticket:  
```
summaryFields: Queue,Status,Subject,Created,Creator
```
as well as aliases for queues you use:  
```
lqn:LongQueueName  
rlqn:ReallyLongQueueName
```
  
### GET FUNCTION  
  
```
Usage: rt get [globaloptions] -t <ticketnum> [options]  
  
  Global Options  
    -config [string]: Alt config file (~/.rt.d/config)  
    -user [string]:   Specify username for Endpoint  
    -e [string]:      Endpoint (ie: my.rtserver.com)  
  
  Options  
    -f: Specify Custom Fields for Summary (csv,list)  
    -t: Ticket Number (required)  
    -p: Show Ticket History  
    -c: Show Ticket Comments  
    -a: List Ticket Attachments  
    -l: List Related Tickets  
    -s: Show Ticket Summary (default)  
```  
The GET function is really just a way to 'read' ticket information when you have a subject ticket. You must specify `-t [int]` to choose a ticket to get information about. If no flags other than -t are chosen, the program will assume you just want a summary. If you only want to see one field you may specify `-f FieldName,Queue` to only show the selected field(s). The `-l` option will make an API call for each related ticket, showing the relationship, ticket number and Subject of Linked tickets.  
  
### SEARCH FUNCTION  
  
```
Usage: rt search [globaloptions] [options]  
  
  Global Options  
    -config [string]: Alt config file (~/.rt.d/config)  
    -user [string]:   Specify username for Endpoint  
    -e [string]:      Endpoint (ie: my.rtserver.com)  
  
  Options  
    -d:          Only show URL (dry run)  
    -o [string]: Query Tickets by Owner (csv,list)  
    -q [string]: Query Tickets by Queue (csv,list)  
    -s [string]: Query Tickets by Status (csv,list)  
    -t [string]: Query Tickets by Title (csv,list)  
      ie: rt search -o jsmith,jdoe -s new,open -q q1,q2  
      ie: rt search -s open -t 'text 1',txt,'text 2'  
```  
The SEARCH function is a quick way to query tickets matching criteria. All criteria is hardcoded to `{Owner or Owner} AND {Queue or Queue} AND {Status or Status} AND {Title or Title}` This will then pull a list of tickets matching specified criteria. As with most flags here, you can specify a comma seperated list to query. Also to note, the `-d` option allows you to just pull the query link (your password will be plain-text!)
  For example: find all tickets matching PROBLEM and owned by john or jack: `rt search -t PROBLEM -o john,jack`  

### CREATE FUNCTION (TODO)
```
This is not yet developed or scoped, Soon to come!
```

### UPDATE FUNCTION (TODO)
Scope of Function:
```
Change Queues
Change Status
Change Priority
Change Ownership (include stealing)
Change Subject
Add Comment
Change Custom Fields
Append Custom Fields
Merge Tickets
Link Tickets
Unlink Tickets
```
