import os
import MySQLdb

def before_all(ctx):
    ctx.debug = ctx.config.userdata.getbool("DEBUG")
    ctx.config = {
        "base": "http://localhost:8080"
    }
    config = {
        'user': 'dbuser',
        'passwd': os.popen('cat /etc/secrets/notify-db.json | grep password | cut -d\'"\' -f4').read().strip(),
        'host': 'notify.cs9ds6yfnikc.us-east-1.rds.amazonaws.com',
        'database': 'notify',
        #'raise_on_warnings': True,
        #'use_pure': False,
    }

    ctx.db = MySQLdb.connect(**config)

def after_all(ctx):
    ctx.db.close()

def after_step(ctx, step):
    if ctx.debug and step.status == "failed":
        import ipdb
        ipdb.post_mortem(step.exc_traceback)

