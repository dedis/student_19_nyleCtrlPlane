from flask import Flask

app = Flask(__name__)


@app.route('/hw2/ex1', methods=['POST'])
def login():
    usr = request.form["user"]
    pas = request.form["pass"]
    pad = "Never send a human to do a machine's job"

    corrpwd = superenc(usr, pad)

    if corrpwd == pas:
        return 200

    return 400


def ascii(a):
    return ord(a[0])


def toChar(i):
    return ''.join(map(unichr, i))


def superenc(msg, key):
    if len(key) < len(msg):
        diff = len(msg) - len(key)
        key += key[0:diff]

    amsg = ascii(msg.split("")[0])
    akey = ascii(key[0:len(msg)].split("")[0])

    x = ""
    for i in range(len(msg)):
        x += toChar(amsg[i] ^ akey[i])

    return base64.b64encode(x)







