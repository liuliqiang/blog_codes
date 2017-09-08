# -*- coding: utf-8 -*-
import email
import imaplib
from email.parser import Parser
from email.header import decode_header
from email.utils import parseaddr


FROM_EMAILS = [
    'webmaster@icbc.com.cn', # ICBC
    'ccsvc@message.cmbchina.com', # CMB
    'creditcard@cgbchina.com.cn', # CGB
]

M = imaplib.IMAP4(IMAP_SERVER)
M.login(FROM_EMAIL, FROM_PWD)
M.select(mailbox='INBOX')
condition = 'ALL'
typ, data = M.search(None, condition)
for num in data[0].split():
    typ, data = M.fetch(num, '(RFC822)')
    try:
        msg = Parser().parsestr(data[0][1].decode('utf-8'))
    except:
        continue
    if parseaddr(msg['From'][1]) in FROM_EMAILS:
        print(parseaddr(msg['From']))
M.close()
M.logout()
