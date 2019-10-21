from manimlib.imports import *
from scipy.spatial import ConvexHull
import random
import numpy as np

random.seed(0)
upgrade_probability = 0.4

def distance(x,y):
    dx = x[0][0] - y[0][0]
    dy = x[0][1] - y[0][1]
    return np.sqrt(dx*dx + dy*dy)


class NodeBunch(Scene):
    def construct(self):
        title = TextMobject("Bunches")
        title.shift(3*UP)
        self.add(title)

        nb_nodes = 20
        system = []
        for i in range(nb_nodes):
            x = random.uniform(-5,4)
            y = random.uniform(-2,1)
            level = 0
            while random.random() < upgrade_probability:
                level += 1


            level_text = TextMobject(str(level))

            system.append(([x, y, 0], level))
            dot = SmallDot(np.array([x, y, 0]), color=WHITE)
            level_text.scale(0.5)
            level_text.next_to(dot, UP)

            self.add(dot)
            self.add(level_text)

        self.wait(2)

        for i in range(5):
            start_node = system[i]
            print(start_node)
            system_sorted = sorted(system, key=lambda x : distance(x,start_node))

            has_seen_level = 0
            for n in system_sorted[1:]:
                if n[1] >= has_seen_level:
                    line = Line(start_node[0],n[0], color=PALETTE[i])
                    line.add_tip(tip_length=0.2)
                    self.play(ShowCreation(line))
                    has_seen_level = n[1]

class NodeCluster(Scene):
    def construct(self):
        title = TextMobject("Clusters")
        title.shift(3*UP)
        self.add(title)

        nb_nodes = 20
        system = []
        clusters = {}
        for i in range(nb_nodes):
            x = random.uniform(-5,4)
            y = random.uniform(-2,1)
            level = 0
            while random.random() < upgrade_probability:
                level += 1


            level_text = TextMobject(str(level))

            elem = ((x, y, 0), level)
            system.append(elem)
            clusters[elem] = []

            dot = SmallDot(np.array([x, y, 0]), color=WHITE)
            level_text.scale(0.5)
            level_text.next_to(dot, UP)
            self.add(dot)
            self.add(level_text)


        for i in range(nb_nodes):
            start_node = system[i]
            system_sorted = sorted(system, key=lambda x : distance(x,start_node))

            has_seen_level = start_node[1]
            for n in system_sorted[1:]:
                if n[1] >= has_seen_level:
                    line = Line(start_node[0],n[0], color=DARK_GREY)
                    clusters[n].append(start_node)
                    line.add_tip(tip_length=0.2)
                    line.set_opacity(0.5)
                    self.add(line)
                    has_seen_level = n[1]

        sorted_clusters = sorted(clusters.items(), key=lambda x : len(x[1]))
        for i, cluster in enumerate(sorted_clusters):
            key = cluster[0]
            print()
            print(cluster)
            print("-----------")
            ##### Show Arrow
            appear_group = []
            disappear_group = []
            for p in clusters[key]:
                line = Line(p[0],key[0])
                line.add_tip(tip_length=0.2)
                appear_group.append(ShowCreation(line))
                disappear_group.append(FadeOut(line))
            if len(appear_group) != 0 :
                self.play(AnimationGroup(*appear_group))

                list_corners = [elem[0] for elem in clusters[key]+[key]]
                if len(list_corners) > 3:
                    hull = [list_corners[v] for v in ConvexHull([[c[0], c[1]] for c in list_corners]).vertices]
                else:
                    hull = list_corners

                ARA = Polygon(*hull, stroke_color=PALETTE[i],stroke_opacity=0.2, fill_color=PALETTE[i], fill_opacity=0.2)
                #ARA.scale(1.5)
                disappear_group.append(ShowCreation(ARA))
                self.play(AnimationGroup(*disappear_group))

        self.wait(2)
