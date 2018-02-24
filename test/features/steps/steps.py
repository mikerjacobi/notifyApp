from behave import *
import json
import requests

use_step_matcher("re")

def wanthave(a, b):
    print("want: %s, have: %s"%(str(a), str(b)))

@step("all test data is cleared")
def clear_test_data(ctx):
    cursor = ctx.db.cursor()
    stmt = "DELETE FROM users WHERE phone_number LIKE '000%'"
    cursor.execute(stmt)
    stmt = "DELETE FROM user_prompts WHERE phone_number LIKE '000%'"
    cursor.execute(stmt)
    stmt = "DELETE FROM communications WHERE to_phone LIKE '000%'"
    cursor.execute(stmt)
    stmt = "DELETE FROM communications WHERE from_phone LIKE '000%'"
    cursor.execute(stmt)
    stmt = "DELETE FROM journals WHERE phone_number LIKE '000%'"
    cursor.execute(stmt)
    ctx.db.commit()

@step('we issue an http (.*) to "(.*)"')
@step('we issue an http (.*) to "(.*)" with data')
def issue_api_call(ctx, method, url):
    headers={"Content-Type": "application/json"}

    payload = '{}'
    if ctx.text:
        json.loads(ctx.text)
        payload = ctx.text

    if method == "POST":
        ctx.resp = requests.post(
            url%ctx.config,
            data=payload,
            headers=headers,
        )
    elif method == "GET":
        ctx.resp = requests.get(
            url%ctx.config,
            headers=headers,
        )
    else:
        raise Exception("method not supported")
@step("we send a text message to the server")
def text_server(ctx):
    payload = {
        "From": ctx.table[0]["from"],
        "Body": ctx.table[0]["message"]
    }

    ctx.resp = requests.post(
        "%(base)s/twilio"%ctx.config,
        data=payload
    )

@step("we receive an http (.*)")
@step("we receive an http (.*) with data")
def check_response(ctx, code):
    have = ctx.resp.status_code
    assert int(code)==have, wanthave(int(code), have)

    if ctx.text:
        want = json.loads(ctx.text)
        have = ctx.resp.json()
        for key in want:
            assert want[key]==have[key], wanthave(want,have)

@step("the (.*) table has data")
def insert_db_data(ctx, table):
    vals = [x for x in ctx.table.rows[0]]
    keys = ",".join(ctx.table.headings)
    params = ",".join(["%s"] * len(vals))
    stmt = "INSERT INTO %s (%s) VALUES (%s)"%(table, keys, params)
    cursor = ctx.db.cursor()
    cursor.execute(stmt, vals)
    ctx.db.commit()

@step("the most recent (.*) row has data like")
def check_db_data(ctx, table):
    table_keys = {
        "communications": ["comms_id", "from_phone", "to_phone", "message", "created"],
        "journals": ["journal_id", "comms_id", "phone_number", "prompt", "entry", "created", "updated"],
    }
    want = json.loads(ctx.text)
    stmt = "SELECT %s FROM %s ORDER BY created DESC LIMIT 1"%(",".join(table_keys[table]), table)
    cursor = ctx.db.cursor()
    cursor.execute(stmt)
    for row in cursor:
        output = dict(zip(table_keys[table], row))
    for key in want:
        assert want[key] in output[key], wanthave(want[key], output[key])
    ctx.db.commit()

