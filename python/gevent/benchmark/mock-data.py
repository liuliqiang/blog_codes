#!/usr/bin/env python
# encoding: utf-8
import mimesis

from app import db, app, User


def encoding(data):
    return bytes(data, "utf-8")

zh = mimesis.Personal('zh')
if __name__ == "__main__":
    with app.app_context():
        db.create_all()

        for i in range(10000000):
            if i % 100000 == 0:
                print(i)
            u = User(username=encoding(zh.username()), email=encoding(zh.email()),
                     first_name=encoding(zh.name()), last_name=encoding(zh.surname()),
                     password=encoding(zh.password()))
            db.session.add(u)
            if i % 1000 == 0:
                try:
                    db.session.commit()
                except:
                    pass
