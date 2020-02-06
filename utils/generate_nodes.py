#! /usr/bin/python
import os
from random import random, seed
import numpy as np

folder_name = [line.strip() for line in open("folder_name")][0]

seed(float(folder_name))

max_size_system = 101
system_width = 300

local_movement = 0.2
teleportation = 0.1

speed_of_light = 299792
distance_to_ping_factor = 1.0/speed_of_light*1000*100

positions = [(random()*system_width, random()*system_width) for _ in range(1,max_size_system)]

def distance(pos1, pos2):
    return np.sqrt((pos1[0]-pos2[0])**2 + (pos1[1]-pos2[1])**2)

print(positions)

for N in range(1,max_size_system):
    for n in range(1,N):
        if random() < local_movement:
            positions[n] = (positions[n][0]+(random()*20-10), positions[n][1]+(random()*20-10))
            print("Epoch : ", N, " local movement for node_",n)
        if random() < teleportation:
            positions[n] = (random()*system_width, random()*system_width)
            print("Epoch : ", N, " teleportation for node_",n)

    with open("NodesFiles/"+str(folder_name)+"/nodes"+str(N)+".txt", "w+") as file_node:
        for i, (x,y) in enumerate(positions):
            if i >= N:
                break
            file_node.write("node_{} {} {}\n".format(i, x,y))

    with open("PingsFiles/"+str(folder_name)+"/pings"+str(N)+".txt", "w+") as file:
        for i in range(N):
            for j in range(i,N):
                r = 0
                if i!=j:
                    r = distance(positions[i], positions[j])*distance_to_ping_factor
                    file.write("ping node_{} node_{} = {} \n".format(i,j, r))
                file.write("ping node_{} node_{} = {}\n".format(j,i, r))
