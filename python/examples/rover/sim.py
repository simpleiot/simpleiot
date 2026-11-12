"""A simulated four-wheel rover and the autonomy that drives it.

Nothing here knows about Simple IoT: rover.py reads commands into
``Commands``, calls ``Rover.step`` on a timer, and reports the rover's state.
To drive real hardware, replace ``Rover`` with something that reads GPS and
sets motor outputs, and keep the rest.

Coordinates are meters on a field 60 m wide and 30 m deep, with (0, 0) at the
bottom left. Heading is in degrees, 0 pointing east and increasing
counterclockwise. A positive steering angle turns left.
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field

FIELD_W = 60.0
FIELD_H = 30.0
HOME = (5.0, 5.0)

# a loop around the field, returning near home
MISSION = [(15, 8), (30, 6), (52, 10), (54, 24), (36, 25), (20, 20), (10, 24), (6, 14)]

WHEELBASE = 1.2  # m
MAX_STEER = 30.0  # degrees
STEER_RATE = 90.0  # degrees per second
ACCEL = 1.0  # m/s per second
ARRIVE = 1.5  # m from a waypoint counts as there

LOW_BATTERY = 20.0  # % at which auto mode goes home to charge
CHARGED = 95.0  # % at which it resumes the mission


@dataclass
class Commands:
    """What people have asked for, as last read from the dashboard."""

    mode: str = "stop"  # stop, manual, auto, or home
    front: bool = True
    rear: bool = True
    speed_limit: float = 2.0  # m/s
    steering: float = 0.0  # degrees, used in manual mode
    estop: bool = False


@dataclass
class Rover:
    x: float = HOME[0]
    y: float = HOME[1]
    heading: float = 90.0
    speed: float = 0.0
    steering: float = 0.0
    battery: float = 100.0
    odometer: float = 0.0
    waypoint: int = 0
    charging: bool = False
    state: str = "stopped"
    mission: list[tuple[float, float]] = field(default_factory=lambda: list(MISSION))

    def step(self, cmd: Commands, dt: float) -> None:
        """Advances the rover by dt seconds under the given commands."""
        target_speed, target_steer = self._decide(cmd)

        # the drives set how much of the limit the rover can reach: both
        # axles give full speed, one gives 60%, and none stops the wheels
        drive = (0.5 if cmd.front else 0.0) + (0.5 if cmd.rear else 0.0)
        drive = 1.0 if drive == 1.0 else 0.6 if drive == 0.5 else 0.0
        target_speed *= drive
        if cmd.estop or drive == 0.0 or self.battery <= 0:
            # stopping the wheels stops the rover at once
            self.speed = 0.0
        else:
            self.speed = _approach(self.speed, target_speed, ACCEL * dt)
        self.steering = _approach(
            self.steering, _clamp(target_steer, -MAX_STEER, MAX_STEER), STEER_RATE * dt
        )

        # bicycle model
        turn = self.speed / WHEELBASE * math.tan(math.radians(self.steering))
        self.heading = (self.heading + math.degrees(turn * dt)) % 360
        h = math.radians(self.heading)
        x = _clamp(self.x + self.speed * math.cos(h) * dt, 0, FIELD_W)
        y = _clamp(self.y + self.speed * math.sin(h) * dt, 0, FIELD_H)
        self.odometer += math.hypot(x - self.x, y - self.y)
        self.x, self.y = x, y

        # driving drains the battery; standing at home charges it
        if self.speed > 0:
            self.battery -= (0.02 + 0.12 * self.speed) * dt
        elif self._at(HOME):
            self.battery += 2.0 * dt
        self.battery = _clamp(self.battery, 0, 100)

    def _decide(self, cmd: Commands) -> tuple[float, float]:
        """Returns the speed and steering the current mode asks for."""
        if cmd.estop:
            self.state = "e-stop"
            return 0.0, self.steering
        if self.battery <= 0:
            self.state = "battery empty"
            return 0.0, self.steering

        if cmd.mode == "manual":
            self.state = "manual"
            return cmd.speed_limit, cmd.steering

        if cmd.mode == "home":
            if self._at(HOME):
                self.state = "home"
                return 0.0, 0.0
            self.state = "returning home"
            return self._drive_to(HOME, cmd.speed_limit)

        if cmd.mode == "auto":
            if self.charging:
                if self.battery >= CHARGED:
                    self.charging = False
                elif self._at(HOME):
                    self.state = f"charging {self.battery:.0f}%"
                    return 0.0, 0.0
                else:
                    self.state = "low battery, returning to charge"
                    return self._drive_to(HOME, cmd.speed_limit)
            if self.battery < LOW_BATTERY:
                self.charging = True
                return self._decide(cmd)
            target = self.mission[self.waypoint]
            if self._at(target):
                self.waypoint = (self.waypoint + 1) % len(self.mission)
                target = self.mission[self.waypoint]
            self.state = f"driving to waypoint {self.waypoint + 1}"
            return self._drive_to(target, cmd.speed_limit)

        self.state = "stopped"
        return 0.0, self.steering

    def _drive_to(self, target: tuple[float, float], limit: float) -> tuple[float, float]:
        """Steers toward a point, slowing for sharp turns and on arrival."""
        dx, dy = target[0] - self.x, target[1] - self.y
        dist = math.hypot(dx, dy)
        error = (math.degrees(math.atan2(dy, dx)) - self.heading + 180) % 360 - 180
        steer = _clamp(error * 1.5, -MAX_STEER, MAX_STEER)
        speed = min(limit, 0.4 + 0.6 * dist) * (0.4 if abs(error) > 60 else 1.0)
        return speed, steer

    def _at(self, p: tuple[float, float]) -> bool:
        return math.hypot(p[0] - self.x, p[1] - self.y) < ARRIVE


def _clamp(v: float, lo: float, hi: float) -> float:
    return max(lo, min(hi, v))


def _approach(v: float, target: float, step: float) -> float:
    return target if abs(target - v) <= step else v + math.copysign(step, target - v)
