#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
古代被中香炉（银熏球）传感器模拟器
模拟传感器上报内环转角、外环转角、炉体倾角、晃荡加速度数据
支持HTTP和MQTT两种上报方式
"""

import json
import math
import random
import time
import argparse
import requests
from datetime import datetime

try:
    import paho.mqtt.client as mqtt
    MQTT_AVAILABLE = True
except ImportError:
    MQTT_AVAILABLE = False


class CenserSimulator:
    def __init__(
        self,
        censer_code,
        api_base="http://localhost:8080/api/v1",
        motion_profile="walking",
        mqtt_broker=None,
        mqtt_topic="censer/sensor",
        mqtt_client_id=None,
        custom_frequency=None,
        custom_amplitude=None,
    ):
        self.censer_code = censer_code
        self.api_base = api_base
        self.motion_profile = motion_profile

        self.inner_angle = 0.0
        self.outer_angle = 0.0
        self.body_tilt = 0.0
        self.slosh_accel = 0.0

        self.inner_vel = 0.0
        self.outer_vel = 0.0
        self.body_ang_vel = 0.0

        self.time = 0.0
        self.temperature = 65.0 + random.uniform(-5, 5)

        self.motion_params = self._get_motion_params(motion_profile)

        if custom_frequency is not None:
            self.motion_params["frequency"] = custom_frequency
            print(f"[模拟器] 自定义颠簸频率: {custom_frequency} Hz")
        if custom_amplitude is not None:
            self.motion_params["amplitude"] = custom_amplitude
            print(f"[模拟器] 自定义颠簸幅度: {custom_amplitude} g")

        self.abnormal_mode = False
        self.abnormal_timer = 0

        self.mqtt_enabled = mqtt_broker is not None
        self.mqtt_broker = mqtt_broker
        self.mqtt_topic = mqtt_topic
        self.mqtt_client_id = mqtt_client_id or f"sim-{censer_code}"
        self.mqtt_client = None

        if self.mqtt_enabled:
            self._init_mqtt()

    def _get_motion_params(self, profile):
        params = {
            "walking": {
                "name": "步行",
                "frequency": 2.0,
                "amplitude": 0.5,
                "tilt_base": 2.0,
                "accel_base": 0.8,
                "noise_level": 0.3,
            },
            "horse_riding": {
                "name": "骑马",
                "frequency": 4.0,
                "amplitude": 2.0,
                "tilt_base": 5.0,
                "accel_base": 2.5,
                "noise_level": 0.8,
            },
            "running": {
                "name": "奔跑",
                "frequency": 6.0,
                "amplitude": 1.5,
                "tilt_base": 4.0,
                "accel_base": 2.0,
                "noise_level": 0.7,
            },
            "car_ride": {
                "name": "乘车",
                "frequency": 8.0,
                "amplitude": 1.0,
                "tilt_base": 3.0,
                "accel_base": 1.5,
                "noise_level": 0.4,
            },
            "sedan_chair": {
                "name": "抬轿",
                "frequency": 1.5,
                "amplitude": 0.8,
                "tilt_base": 3.5,
                "accel_base": 1.0,
                "noise_level": 0.5,
            },
            "static": {
                "name": "静止",
                "frequency": 0.1,
                "amplitude": 0.05,
                "tilt_base": 0.2,
                "accel_base": 0.05,
                "noise_level": 0.05,
            },
            "violent": {
                "name": "剧烈颠簸",
                "frequency": 10.0,
                "amplitude": 5.0,
                "tilt_base": 10.0,
                "accel_base": 6.0,
                "noise_level": 1.5,
            },
        }
        return params.get(profile, params["walking"])

    def _init_mqtt(self):
        if not MQTT_AVAILABLE:
            print("[警告] paho-mqtt 未安装，MQTT模式不可用")
            self.mqtt_enabled = False
            return

        self.mqtt_client = mqtt.Client(self.mqtt_client_id)
        self.mqtt_client.on_connect = self._on_mqtt_connect
        self.mqtt_client.on_disconnect = self._on_mqtt_disconnect

        try:
            host, port = self._parse_broker(self.mqtt_broker)
            self.mqtt_client.connect_async(host, port, keepalive=60)
            self.mqtt_client.loop_start()
            print(f"[MQTT] 正在连接到 {self.mqtt_broker}...")
        except Exception as e:
            print(f"[MQTT] 连接失败: {e}")
            self.mqtt_enabled = False

    def _parse_broker(self, broker):
        if "://" in broker:
            broker = broker.split("://")[1]
        if ":" in broker:
            parts = broker.split(":")
            return parts[0], int(parts[1])
        return broker, 1883

    def _on_mqtt_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"[MQTT] 已连接到 broker: {self.mqtt_broker}")
        else:
            print(f"[MQTT] 连接失败，错误码: {rc}")

    def _on_mqtt_disconnect(self, client, userdata, rc):
        print(f"[MQTT] 连接断开，错误码: {rc}")

    def set_motion_profile(self, profile):
        self.motion_profile = profile
        self.motion_params = self._get_motion_params(profile)
        print(f"[模拟器] 切换运动模式: {self.motion_params['name']}")

    def set_custom_params(self, frequency=None, amplitude=None):
        if frequency is not None:
            self.motion_params["frequency"] = frequency
            print(f"[模拟器] 设置颠簸频率: {frequency} Hz")
        if amplitude is not None:
            self.motion_params["amplitude"] = amplitude
            print(f"[模拟器] 设置颠簸幅度: {amplitude} g")

    def trigger_abnormal(self, duration=30):
        self.abnormal_mode = True
        self.abnormal_timer = duration
        print(f"[模拟器] 触发异常工况 {duration}秒...")

    def step(self, dt=60):
        self.time += dt
        p = self.motion_params

        omega = 2 * math.pi * p["frequency"]
        t = self.time

        if self.abnormal_mode:
            self.abnormal_timer -= 1
            if self.abnormal_timer <= 0:
                self.abnormal_mode = False
                print("[模拟器] 异常工况结束，恢复正常")
            tilt_mult = 3.5
            accel_mult = 3.0
            noise_mult = 2.5
        else:
            tilt_mult = 1.0
            accel_mult = 1.0
            noise_mult = 1.0

        target_inner = (
            p["amplitude"] * 20 * math.sin(omega * t + random.uniform(-0.2, 0.2))
            + random.gauss(0, p["noise_level"] * 5) * noise_mult
        )
        target_outer = (
            p["amplitude"] * 25 * math.sin(omega * t * 0.7 + math.pi / 4 + random.uniform(-0.2, 0.2))
            + random.gauss(0, p["noise_level"] * 6) * noise_mult
        )
        target_tilt = (
            p["tilt_base"] * tilt_mult + 8 * tilt_mult * abs(math.sin(omega * t * 0.5))
            + random.gauss(0, p["noise_level"] * 3) * noise_mult
        )
        target_accel = (
            p["accel_base"] * accel_mult * (1 + 0.6 * abs(math.sin(omega * t)))
            + random.gauss(0, p["noise_level"]) * noise_mult
        )

        alpha = 0.3
        self.inner_angle = self.inner_angle * (1 - alpha) + target_inner * alpha
        self.outer_angle = self.outer_angle * (1 - alpha) + target_outer * alpha
        self.body_tilt = max(0, self.body_tilt * (1 - alpha) + target_tilt * alpha)
        self.slosh_accel = max(0, self.slosh_accel * (1 - alpha) + target_accel * alpha)

        self.inner_vel = (target_inner - self.inner_angle) / dt
        self.outer_vel = (target_outer - self.outer_angle) / dt
        self.body_ang_vel = (target_tilt - self.body_tilt) / dt

        self.temperature += random.uniform(-0.3, 0.5)
        self.temperature = max(50, min(95, self.temperature))

        return {
            "censer_code": self.censer_code,
            "inner_ring_angle": round(self.inner_angle, 4),
            "outer_ring_angle": round(self.outer_angle, 4),
            "body_tilt": round(self.body_tilt, 4),
            "slosh_acceleration": round(self.slosh_accel, 4),
            "inner_ring_velocity": round(self.inner_vel, 4),
            "outer_ring_velocity": round(self.outer_vel, 4),
            "body_angular_velocity": round(self.body_ang_vel, 4),
            "temperature": round(self.temperature, 2),
            "timestamp": int(time.time()),
        }

    def send_data_http(self, data):
        url = f"{self.api_base}/sensor-data"
        try:
            resp = requests.post(url, json=data, timeout=5)
            if resp.status_code in (200, 201):
                result = resp.json()
                return True, result
            else:
                print(f"[HTTP错误] {resp.status_code}: {resp.text[:100]}")
                return False, None
        except requests.exceptions.RequestException as e:
            print(f"[HTTP错误] 连接失败: {e}")
            return False, None

    def send_data_mqtt(self, data):
        if not self.mqtt_enabled or not self.mqtt_client:
            return False

        try:
            topic = f"{self.mqtt_topic}/{self.censer_code}"
            payload = json.dumps(data)
            result = self.mqtt_client.publish(topic, payload, qos=1)
            return result.rc == 0
        except Exception as e:
            print(f"[MQTT错误] 发布失败: {e}")
            return False

    def send_data(self, data):
        http_ok = False
        http_result = {}

        if self.api_base:
            http_ok, http_result = self.send_data_http(data)

        mqtt_ok = False
        if self.mqtt_enabled:
            mqtt_ok = self.send_data_mqtt(data)

        ts = datetime.now().strftime("%H:%M:%S")
        status_icon = "�"
        spill_risk = http_result.get("spill_risk", 0) if http_ok else 0
        balance_score = http_result.get("balance_score", 0) if http_ok else 0

        if spill_risk > 0.5:
            status_icon = "🔴"
        elif spill_risk > 0.2:
            status_icon = "🟡"

        mode_str = ""
        if self.api_base and self.mqtt_enabled:
            mode_str = " [HTTP+MQTT]"
        elif self.mqtt_enabled:
            mode_str = " [MQTT]"
        else:
            mode_str = " [HTTP]"

        print(
            f"[{ts}]{mode_str} {self.censer_code} | "
            f"内环:{data['inner_ring_angle']:>+7.2f}° "
            f"外环:{data['outer_ring_angle']:>+7.2f}° "
            f"倾角:{data['body_tilt']:>6.2f}° "
            f"加速度:{data['slosh_acceleration']:>5.2f}m/s² "
            f"| 平衡:{balance_score:.3f} "
            f"风险:{spill_risk:.3f} "
            f"{status_icon}"
        )

        return http_ok or mqtt_ok

    def run(self, interval=60, stop_after=None):
        print(f"=" * 70)
        print(f"  被中香炉传感器模拟器启动")
        print(f"  设备编号: {self.censer_code}")
        if self.api_base:
            print(f"  API地址:  {self.api_base}")
        if self.mqtt_enabled:
            print(f"  MQTT Broker: {self.mqtt_broker}")
            print(f"  MQTT Topic:  {self.mqtt_topic}/{self.censer_code}")
        print(f"  运动模式: {self.motion_params['name']}")
        print(f"  颠簸频率: {self.motion_params['frequency']} Hz")
        print(f"  颠簸幅度: {self.motion_params['amplitude']} g")
        print(f"  上报间隔: {interval}秒")
        print(f"=" * 70)
        print()

        count = 0
        try:
            while True:
                data = self.step(interval)
                self.send_data(data)
                count += 1

                if stop_after and count >= stop_after:
                    print(f"\n[模拟器] 已完成 {count} 次上报，停止运行")
                    break

                time.sleep(interval)

        except KeyboardInterrupt:
            print(f"\n[模拟器] 用户中断，共上报 {count} 次数据")
        finally:
            if self.mqtt_client:
                self.mqtt_client.loop_stop()
                self.mqtt_client.disconnect()

    def cleanup(self):
        if self.mqtt_client:
            self.mqtt_client.loop_stop()
            self.mqtt_client.disconnect()


def main():
    parser = argparse.ArgumentParser(description="被中香炉传感器模拟器")
    parser.add_argument("-c", "--censer", default="CENSER-001",
                        help="香炉设备编号 (默认: CENSER-001)")
    parser.add_argument("-a", "--api", default="http://localhost:8080/api/v1",
                        help="后端API地址，设为空字符串则禁用HTTP")
    parser.add_argument("-i", "--interval", type=float, default=60,
                        help="上报间隔秒数 (默认: 60)")
    parser.add_argument("-m", "--motion",
                        choices=["walking", "horse_riding", "running", "car_ride",
                                 "sedan_chair", "static", "violent"],
                        default="walking", help="运动模式 (默认: walking)")
    parser.add_argument("-n", "--number", type=int, default=None,
                        help="运行指定次数后停止 (默认: 无限)")
    parser.add_argument("--fast", action="store_true",
                        help="快速模式: 1秒间隔，用于演示")

    parser.add_argument("--mqtt-broker", default=None,
                        help="MQTT Broker地址 (如: mqtt://localhost:1883)")
    parser.add_argument("--mqtt-topic", default="censer/sensor",
                        help="MQTT主题前缀 (默认: censer/sensor)")
    parser.add_argument("--mqtt-only", action="store_true",
                        help="仅使用MQTT上报，禁用HTTP")

    parser.add_argument("--frequency", type=float, default=None,
                        help="自定义颠簸频率 (Hz)，覆盖运动模式默认值")
    parser.add_argument("--amplitude", type=float, default=None,
                        help="自定义颠簸幅度 (g)，覆盖运动模式默认值")

    parser.add_argument("--multi", type=int, default=1,
                        help="同时模拟多个香炉设备 (1-3)")

    args = parser.parse_args()

    if args.fast:
        args.interval = 1

    api_base = args.api
    if args.mqtt_only:
        api_base = ""

    mqtt_broker = args.mqtt_broker

    censer_codes = ["CENSER-001", "CENSER-002", "CENSER-003"]
    simulators = []

    for i in range(min(args.multi, 3)):
        sim = CenserSimulator(
            censer_code=censer_codes[i],
            api_base=api_base,
            motion_profile=args.motion,
            mqtt_broker=mqtt_broker,
            mqtt_topic=args.mqtt_topic,
            mqtt_client_id=f"sim-{censer_codes[i]}",
            custom_frequency=args.frequency,
            custom_amplitude=args.amplitude,
        )
        simulators.append(sim)

    if len(simulators) == 1:
        simulators[0].run(interval=args.interval, stop_after=args.number)
    else:
        print(f"[模拟器] 同时模拟 {len(simulators)} 个香炉设备")
        count = 0
        try:
            while True:
                for sim in simulators:
                    data = sim.step(args.interval)
                    sim.send_data(data)
                count += 1

                if args.number and count >= args.number:
                    break

                time.sleep(args.interval)
        except KeyboardInterrupt:
            print(f"\n[模拟器] 用户中断，共上报 {count} 轮")
        finally:
            for sim in simulators:
                sim.cleanup()


if __name__ == "__main__":
    main()
