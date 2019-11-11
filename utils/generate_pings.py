import os
from random import random

N = 10

with open("pings.txt", "w") as file:
    for i in range(N):
        for j in range(i,N):
            r = 0
            if i!=j:
                r = random()*40
                file.write("ping node_{} node_{} = {} \n".format(i,j, r))
            file.write("ping node_{} node_{} = {}\n".format(j,i, r))
