Feature: create and register account
    Scenario: postive
        Given all test data is cleared
        When we issue an http POST to "%(base)s/twirp/notify.NotifyApp/CreateAccount" with data
        """
        {
          "user": {
            "phone_number": "0004451322",
            "password": "abcdef",
            "name": "mike",
            "birthday": "1989-07-04"
          },
          "password_repeat": "abcdef"
        }
        """
        Then we receive an http 200 with data
        """
        {"success": true}
        """
        And the most recent communications row has data like
        """
        {"message": "respond with 'reg'"}
        """

    Scenario Outline: negative
        Given all test data is cleared
        When we issue an http POST to "%(base)s/twirp/notify.NotifyApp/CreateAccount" with data
        """
        {
          "user": {
            "phone_number": "<phone_number>",
            "password": "<password>",
            "name": "<name>",
            "birthday": "<birthday>"
          },
          "password_repeat": "<repeat>"
        }
        """
        Then we receive an http 400
        Examples:
            | phone_number  | password | name | birthday   | repeat |
            | 1             | abcdef   | mike | 1989-07-04 | abcdef |
            | 0004254451322 |          | mike | 1989-07-04 | abcdef |
            | 0004254451322 | abcdef   |      | 1989-07-04 | abcdef |
            | 0004251322    | abcdef   | mike | 19890704   | abcdef |
            | 0004251322    | abcdef   | mike | 1989-07-04 | acdef  |
