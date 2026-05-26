## **Local DevOps Without Internet**

**Use case**: A room full of Raspberry Pis or dev boards at a workshop

* Flash SD cards, boot up, and just run:

  ```
  meshexec run --target="arch=armv7" "git clone https://repo && ./install.sh"
  ```

**Effect**: Devices discover each other (mDNS) and self‑provision. No central server.

---

## **Disaster Relief / Off-Grid Ops**

**Use case**: No Wi-Fi, no LTE, no satellites. You and your crew have laptops and sensors.

* Setup a field command device and send:

  ```
  meshexec run --target="drone" "collect-temp && send-report"
  ```

**Effect**: Devices on the same LAN coordinate without a central server.

---

## **Smart Conference Badges**

**Use case**: Conference badges with BLE and small CPUs

* Trigger LEDs, update firmware, or send "game" states:

  ```
  meshexec run --target="badge && role=volunteer" "led blink 3"
  ```

**Effect**: Mass control on the same LAN; no IP sharing needed.

---

## **Classroom or Exam Hall Control**

**Use case**: A teacher with 30 Raspberry Pis or tablets

* At once:

  ```
  meshexec run --target="student" "open quiz.html"
  ```

* Or later:

  ```
  meshexec run "cat answers.txt" > all_answers/
  ```

**Effect**: Distributed control over an entire room, no need to micromanage devices individually.

---

## **Drone/Robot Swarm Management**

**Use case**: Multiple ESP32/robot units in the field

* Command all:

  ```
  meshexec run --target="robot && zone=alpha" "move_to 32.12 -81.23"
  ```

* Or issue a synchronized action:

  ```
  meshexec run --target="robot" --sync "start_dance_mode"
  ```

**Effect**: Choreographed action. Mesh allows indirect control of units out of range.

---

## **Scientific Field Kits**

**Use case**: A network of sensors in the jungle/lab/mine

* Command them to calibrate, start measurements, or dump logs:

  ```
  meshexec run --target="sensor && type=CO2" "calibrate"
  ```

**Effect**: No centralized hub needed. You can command everything from your phone or laptop.

---

## **LAN-Party Controller**

**Use case**: Control a local gaming session with friends

* Start game sessions, send trash talk, auto-setup mod configs:

  ```
  meshexec run --target="gamer" "start quake3"
  ```

**Effect**: You become the party master. You don’t need to tell anyone what IP to use.

---

## **Git Pull Party**

**Use case**: Sync local clones of a repo between friends offline

* One person is the source:

  ```
  meshexec sync --repo .
  ```

* Others run:

  ```
  meshexec clone
  ```

**Effect**: BLE-powered ad-hoc version control when you're cut off from GitHub.

---

## **BioLab Mesh Orchestrator**

**Use case**: Lab full of microcontrollers controlling centrifuges, sensors, heaters

* One command:

  ```
  meshexec run --target="device=centrifuge" "spin 1000rpm 10min"
  ```

**Effect**: Orchestrate whole lab protocols across LAN‑connected nodes.
