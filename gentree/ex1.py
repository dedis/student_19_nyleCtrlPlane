from flask import Flask

from flask import render_template
from flask import request

import base64
import sys
import logging

logging.basicConfig(stream=sys.stderr, level=logging.DEBUG)

app = Flask(__name__)
@app.route('/')


@app.route('/hw2/ex1', methods=['POST'])
def ex1():
    try:
        if request.method == 'POST':
            req = request.json
            usr = req["user"]
            pas = req["pass"]
            pad = "Never send a human to do a machine's job"


            corrpwd = superenc(usr, pad)
            print(corrpwd, file=sys.stderr)
            print(pas, file=sys.stderr)

    except Exception as e:
        logging.exception(e)


    if corrpwd == pas:
        return "",200

    return "",400



def toChar(i):
    print(int(i))
    return chr(int(i))


def superenc(msg, key):
    if len(key) < len(msg):
        diff = len(msg) - len(key)
        key += key[0:diff]

    x = ""
    akey = [ord(i) for i in str(msg)]
    amsg = [ord(i) for i in str(key[0:len(msg)])]

    for i in range(len(akey)):
        x += toChar(akey[i] ^ amsg[i])

    return base64.b64encode(x.encode('ascii')).decode("utf-8")


app.run(debug=True)