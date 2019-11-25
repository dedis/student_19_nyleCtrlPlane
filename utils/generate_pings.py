import os
from random import random

for N in range(1,50):
    with open("PingsFiles/pings"+str(N)+".txt", "w+") as file:
        for i in range(N):
            for j in range(i,N):
                r = 0
                if i!=j:
                    r = random()*40
                    file.write("ping node_{} node_{} = {} \n".format(i,j, r))
                file.write("ping node_{} node_{} = {}\n".format(j,i, r))
