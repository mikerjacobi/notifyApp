Feature: respond to prompt
    Scenario: postive
        Given all test data is cleared
        Given the users table has data
            | phone_number | name | birthday   | hashword                                                                         | verified | created                    | updated                    |
            | 0005551234   | mike | 1989-07-04 | JDJhJDEwJERodnJnR2t1Y1AuaWJwazdTQUZPR2V1R2FoS2ljemFWT2UzZkpndkMxTmFRaVNaaU00Zm5x | 1        | 2018-01-30 03:49:55.971300 | 2018-01-30 03:49:55.971300 |
        When we issue an http POST to "%(base)s/twirp/notify.NotifyApp/AddUserPrompt" with data
        """
        {
            "prompt_id": "7b1ced70-a2a0-40c5-8aa5-1cc5cff3b04b",
            "phone_number": "0005551234",
            "frequency": "24h",
            "next_prompt_time": "2018-01-29 20:30:00"
        }
        """
        Then we receive an http 200
        When we issue an http POST to "%(base)s/twirp/notify.NotifyApp/TriggerNotifications"
        Then we receive an http 200
        And the most recent communications row has data like
        """
        {
            "to_phone": "0005551234",
            "message": "What did you have for lunch?"
        }
        """
        When we send a text message to the server
            | from       | message     |
            | 0005551234 | hello world |
        Then we receive an http 200
        And the most recent journals row has data like
        """
        {
            "phone_number": "0005551234",
            "prompt": "What did you have for lunch?",
            "entry": "hello world"
        }
        """
