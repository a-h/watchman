# Observer

Observes Github repositories to look for issues and commits which may be security vulnerabilities, notifying to a Slack channel.

# Usage

* Configure your target SNS notification endpoint in `serverless.yml`.
* Install the app to your AWS environment using the Serverless Framework's `serverless deploy` command.
  * `serverless deploy --alertSnsTopicArn=<TOPIC_ARN> --githubToken=<GITHUB_TOKEN>`

# Structure

* The `/start` directory contains a Lambda which is triggered by a timer. It retrieves a list of repositories to query from the database, and places a message for each one on the repo input queue. At the end of the Lambda, the lastUpdated field on each repo is updated to the current time.
* The `/repo` directory contains a Lambda triggered by a message on the repo input queue. It retrieves a list of issues on the input repo and, for each issues which has been updated since the last time a check was made, places a message on the issue input queue.
* The `/issue` directory contains a Lambda triggered by a message on the issue input queue. It retrieves a list of comments on the input issue, and for each comment which has been updated since the last time a check was made, analyses the comment for keywords relating to security. If any of the keywords match, a notification is placed on the configured SNS topic.
