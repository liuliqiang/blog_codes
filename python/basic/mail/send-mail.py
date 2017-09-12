# -*- coding: utf-8 -*-
from smtplib import SMTP_SSL as SMTP
import logging
import logging.handlers
import sys
from email.mime.text import MIMEText

def send_confirmation():
    text = '''
    Hello,

    Here is your test email.

    Cheers!
    '''
    msg = MIMEText(text, 'plain')
    msg['Subject'] = "test email"
    me = FROM_EMAIL
    msg['To'] = me
    try:
        conn = SMTP(SMTP_SERVER)
        conn.set_debuglevel(True)
        conn.login(FROM_EMAIL, FROM_PWD)
        try:
            conn.sendmail(me, me, msg.as_string())
        finally:
            conn.close()

    except Exception as exc:
        print("ERROR!!!")
        print(exc)
        sys.exit("Mail failed: {}".format(exc))


if __name__ == "__main__":
    send_confirmation()
