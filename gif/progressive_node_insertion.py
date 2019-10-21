from manimlib.imports import *
import random
import numpy as np

random.seed(2)
upgrade_probability = 0.25
final_n_nodes = 20
random.random()


class Region:
    def __init__(self, scene, system, nodes_id, color=WHITE):
        self.nodes_id = nodes_id
        self.representation = None
        self.color = color
        self.draw_init(scene, system)


    def add(self, node):
        self.nodes_id.append(node.id)


    def update_representation(self, system):
        if len(self.nodes_id) > 1:
            list_corners = [system[id].pos()
                            for id in self.nodes_id]
            if len(list_corners) > 3:
                hull = [list_corners[v] for v in ConvexHull(
                    [[c[0], c[1]] for c in list_corners]).vertices]
            else:
                hull = list_corners

            ARA = Polygon(
                *hull,
                stroke_color=self.color,
                stroke_opacity=0.2,
                fill_color=self.color,
                fill_opacity=0.2)
            self.representation = ARA

    def draw_init(self, scene, system):
        self.update_representation(system)
        if self.representation:
            scene.play(ShowCreation(self.representation))

    def draw_extend(self, scene, system):
        old_rep = None if self.representation == None else self.representation
        self.update_representation(system)
        if self.representation:
            if old_rep == None:
                self.draw_init(scene,system)
            else:
                scene.play(ReplacementTransform(old_rep, self.representation))


    def points(self):
        return sorted(self.nodes_id)




class Node:
    def __init__(self, id, x=None, y=None, level=0, color=WHITE):
        self.id = id
        self.x = x if x is not None else random.uniform(-5, 4)
        self.y = y if y is not None else random.uniform(-2, 1)
        self.level = level
        self.level_text = TextMobject(str(self.level))
        self.color = color
        self.dot = SmallDot(np.array([self.x, self.y, 0]), color=self.color)
        self.bunch = set()
        self.bunch_arrows = {}
        self.cluster = set()
        self.regions = set()

    def draw(self, scene):
        self.level_text.scale(0.5)
        self.level_text.next_to(self.dot, UP)
        write_dot_group = [ShowCreation(self.dot), Write(self.level_text)]
        scene.play(AnimationGroup(*write_dot_group))

    def update_level(self, scene):
        old_text = self.level_text
        self.level_text = TextMobject(str(self.level))
        self.level_text.scale(0.5)
        self.level_text.next_to(self.dot, UP)
        scene.play(ReplacementTransform(old_text, self.level_text))

    def distance(self, node):
        dx = self.x-node.x
        dy = self.y-node.y
        return np.sqrt(dx*dx + dy*dy)

    def pos(self):
        return [self.x, self.y, 0]

    def compute_bunch(self, system):
        system_sorted = sorted(
            system, key=lambda x: self.distance(x))

        has_seen_level = self.level
        self.bunch = set()
        for n in system_sorted[1:]:
            if n.level >= has_seen_level:
                self.bunch.add(n.id)
                system[n.id].cluster.add(self.id)
                has_seen_level = n.level
            elif self.id in system[n.id].cluster:
                system[n.id].cluster.remove(self.id)

        return self.bunch

    def draw_bunch(self, scene, system, color=WHITE):
        appear_arrows = []
        bunch = self.compute_bunch(system)
        if len(bunch) != 0:
            for id in self.compute_bunch(system):
                line = Line(self.pos(), system[id].pos(), color=color)
                line.add_tip(tip_length=0.2)
                faded_line = line.copy()
                faded_line.set_opacity(0.1)
                self.bunch_arrows[id] = faded_line
                appear_arrows.append(Succession(
                    ShowCreation(line), Transform(line, faded_line)))
            scene.play(AnimationGroup(*appear_arrows))

    def draw_and_update_bunch(self, scene, system):
        old_bunch = self.bunch.copy()
        new_bunch = self.compute_bunch(system)

        appear_arrows = []
        for id in new_bunch-old_bunch:
            line = Line(self.pos(), system[id].pos(), color=WHITE)
            line.add_tip(tip_length=0.2)
            faded_line = line.copy()
            faded_line.set_opacity(0.1)
            self.bunch_arrows[id] = faded_line
            appear_arrows.append(Succession(
                ShowCreation(line), Transform(line, faded_line)))

        for id in old_bunch-new_bunch:
            print("Erase !")
            line = Line(self.pos(), system[id].pos(), color=RED)
            line.add_tip(tip_length=0.2)
            scene.remove(self.bunch_arrows[id])
            appear_arrows.append(Succession(
                ShowCreation(line), FadeOut(line)))

        return appear_arrows

    def points(self):
        return sorted(list(self.cluster)+[self.id])

    def join_or_create_regions(self,scene, system, region_list):
        for n in self.bunch:
            for region_id in system[n].regions:
                region = region_list[region_id]
                region.add(self)
                region.draw_extend(scene, system)

        if all(r.points() != self.points() for r in region_list) or region_list == []:
            new_region = Region(scene,system, self.points(), color=PALETTE[len(region_list)])
            region_list.append(new_region)
            print("Region Creation, n regions : ", len(region_list))
            self.regions.add(len(region_list)-1)



def update_bunches(scene, system):
    appear_arrows = []
    for n in system:
        appear_arrows.extend(n.draw_and_update_bunch(scene, system))
    if len(appear_arrows) > 0:
        scene.play(AnimationGroup(*appear_arrows))

def election(scene, system, level):
    candidates = [n for n in system if n.level == level]
    winner = random.choice(candidates)
    result = random.random()
    if result < upgrade_probability:
        print("Winner :", winner.id, " - Level ", level+1, " with result : ", result)
        winner.level += 1
        winner.update_level(scene)
        election(scene, system, level+1)



class ProgressiveJoin(Scene):
    CONFIG = {
    "random_seed" : 1
    }
    def construct(self):
        title = TextMobject("Progressive Join")
        title.shift(3*UP)
        self.add(title)

        system = []

        region_list = []
        for i in range(final_n_nodes):
            node = Node(i)
            node.draw(self)
            system.append(node)
            election(self,system, 0)
            node.draw_bunch(self, system)
            update_bunches(self, system)
            node.join_or_create_regions(self, system, region_list)



        self.wait(2)
