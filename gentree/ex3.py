from flask import request
from flask import Flask
from flask import make_response
import time
import hmac

MYEMAIL = "lef@epfl.ch"
otp = "Never send a human to do a machine's job"
cookie_name = "LoginCookie"

import base64
app = Flask(__name__)
@app.route('/')




@app.route("/ex3/login", methods=['POST'])
def ex3():
    user = request.json["user"]
    pwd = request.json["pass"]
    if user == "administrator" and pwd == "42":
        flag = True
    else:
        flag = False
    c = cookie(user,flag)
    resp = make_response()
    resp.set_cookie(cookie_name,c)
    return resp

@app.route("/ex3/list",methods=['POST'])
def ex3_list():
    cookie = request.cookies[cookie_name]
    if is_valid_cookie(cookie):
        if is_admin_cookie(cookie):
            return "",200
        else:
            return "",201
        # return unauthorized status
    return "",403


def is_valid_cookie(cookie):
    generator = hmac.new(superencryption(MYEMAIL,otp))
    infos = base64.b64decode(cookie).decode('utf-8').split(",")
    temp = ",".join([infos[0],infos[1],infos[2],infos[3],infos[4],infos[5]]).encode()
    generator.update(temp)
    print(infos[6])
    print(generator.hexdigest())
    return infos[6] == generator.hexdigest()

def is_admin_cookie(cookie):
    infos = base64.b64decode(cookie).decode('utf-8').split(",")
    return len(infos) == 7 and infos[5] == "administrator"


def cookie(user,flag):
    generator = hmac.new(superencryption(MYEMAIL,otp))
    t = str(int(time.time()))
    domain = "com402"
    hw = "hw2"
    ex = "ex3"
    if flag:
        role = "administrator"
    else:
        role = "user"
    cookie1 = ",".join([user,t,domain,hw,ex,role]).encode()
    generator.update(cookie1)
    cookie = ",".join([user,t,domain,hw,ex,role,generator.hexdigest()]).encode()
    return base64.b64encode(cookie).decode('utf-8')

def superencryption(msg,key):
    if len(key) < len(msg):
        diff = len(msg) - len(key)
        key += key[0:diff]

    mmap = map(ord,msg)
    kmap = map(ord,key)
    xored = map(lambda x,y: x^y,mmap,kmap)
    return base64.b64encode(bytes(xored))

app.run(debug=True)