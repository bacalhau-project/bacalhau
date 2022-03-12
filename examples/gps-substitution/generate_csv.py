from itertools import count
from math import asin, sqrt, sin, cos, atan2
import datetime, time
from random import randint, random, randrange
from pathlib import Path
import csv
from typing import Counter
import pandas as pd
from pandas import DataFrame
import numpy as np
from numpy import deg2rad, float64, savetxt
import pandas
from scipy import rand, stats
import math


def main():
    EARTH_RADIUS = 6372
    NUMBER_OF_ENTRIES = 1000
    DISTANCE_TO_FILTER_IN_KM = 25
    # start_date = datetime.datetime.now()
    START_DATE = datetime.datetime(2021, 1, 1, 0, 0, 0)
    PEAK_DAY = datetime.datetime(2021, 6, 22)

    class GPSPoint:
        def __init__(self, lat: float, long: float):
            self.lat = lat
            self.long = long

    class City:
        def __init__(self, lat: float, long: float, average_peak_temp: float, average_low_temp: float, day_variation: float):
            self.center = GPSPoint(lat, long)
            self.average_peak_temp = average_peak_temp
            self.average_low_temp = average_low_temp
            self.day_variation = day_variation

    class Entry:
        def __init__(
            self,
            date,
            lat,
            long,
            value,
        ) -> None:
            pass

    headers = ["sensor_time", "sensor_group", "lat", "long", "temperature", "distance"]

    def distanceInKmBetweenEarthCoordinates(lat1, lon1, lat2, lon2) -> float:
        earthRadiusKm = 6371

        dLat = deg2rad(lat2 - lat1)
        dLon = deg2rad(lon2 - lon1)

        lat1 = deg2rad(lat1)
        lat2 = deg2rad(lat2)

        # a = sin(dLat / 2) * sin(dLat / 2) + sin(dLon / 2) * sin(dLon / 2) * cos(lat1) * cos(lat2)
        a = np.sin(dLat / 2)
        b = np.sin(dLat / 2)
        c = np.sin(dLon / 2)
        d = np.sin(dLon / 2)
        e = cos(lat1)
        f = np.cos(lat2)

        g = a * b + c * d * e * f
        c = 2 * np.arctan2(np.sqrt(g), np.sqrt(1 - g))
        return earthRadiusKm * c

    def calc_temperature(current_time: np.datetime64, city: City) -> float:
        days_since_jan_1 = current_time.astype("datetime64[Y]") - current_time.astype("datetime64[D]")
        minutes_since_midnight = current_time.astype("datetime64[D]") - current_time.astype("datetime64[m]")
        daily_peak_temp = np.array(city.average_peak_temp - ((city.average_peak_temp - city.average_low_temp) * (abs(days_since_jan_1 + 180).astype(int) / 180)))
        minute_temp = daily_peak_temp - (city.day_variation * (abs(minutes_since_midnight.astype(float64) + 12 * 60)) / (12 * 60))

        # https://stackoverflow.com/questions/28643993/get-random-numbers-within-one-standard-deviation
        # temp_array = stats.truncnorm.rvs(-2, 2, loc=minute_temp, scale=4, size=1)
        return np.random.normal(minute_temp.astype(np.float64), 0.5)

    cities = {
        "NEW_YORK": City(40.7127281, -74.0060152, 30, 4, 9),
        "MUMBAI": City(19.0759899, 72.8773928, 34, 18, 12),
        "LISBON": City(38.7077507, -9.1365919, 28, 15, 9),
    }

    df = DataFrame(index=range(NUMBER_OF_ENTRIES * len(cities)), columns=headers)
    all_cities_array = np.array([], dtype=object).reshape(0, len(headers))

    class DistanceCounter:
        def __init__(self):
            self.distance_from_greater = 0
            self.distance_from_less = 0

        def increment_distance(self, distance):
            if distance > DISTANCE_TO_FILTER_IN_KM:
                self.distance_from_greater += 1
            else:
                self.distance_from_greater += 1

    class CityCounter:
        def __init__(self):
            self.cities = {}

        def increment_distance(self, city_name, distance):
            if city_name not in self.cities:
                self.cities[city_name] = DistanceCounter()

            self.cities[city_name].increment_distance(distance)

    counters = CityCounter()

    idx = 0
    # idx = len(cities) * i
    # if (idx % 10000) == 0:
    #     print(f"Iteration: {idx}")

    for city_name, city in cities.items():
        nparray = np.zeros(shape=(NUMBER_OF_ENTRIES, len(headers)), dtype=object)
        nparray[:, 0] = np.arange(
            START_DATE,
            START_DATE + datetime.timedelta(minutes=NUMBER_OF_ENTRIES),
            datetime.timedelta(minutes=1),
            dtype="M8[m]",
        )

        nparray[:, 1] = np.full(shape=(NUMBER_OF_ENTRIES), fill_value=city_name)

        nparray[:, 2] = np.random.normal(float(city.center.lat), 0.25, size=NUMBER_OF_ENTRIES)
        nparray[:, 3] = np.random.normal(float(city.center.long), 0.25, size=NUMBER_OF_ENTRIES)

        nparray[:, 4] = calc_temperature(nparray[:, 0].astype(np.datetime64), city)

        nparray[:, 5] = distanceInKmBetweenEarthCoordinates(city.center.lat, city.center.long, nparray[:, 2].astype(float64), nparray[:, 3].astype(float64))

        all_cities_array = np.concatenate([all_cities_array, nparray])

    # print(f"Greater: {counters.greater}\nLess: {counters.less}")

    df = pd.DataFrame(data=all_cities_array, columns=headers)

    print(
        f"""
    Total Samples: {len(df)}
    Total from Lisbon: {len(df[df.sensor_group == "LISBON"])}
    Total within {DISTANCE_TO_FILTER_IN_KM} km of LISBON city center: {len(df[(df.sensor_group == "LISBON") & (df.distance < DISTANCE_TO_FILTER_IN_KM)])}
    """
    )

    savetxt(str(Path(__file__).parent / "temperature_sensor_data.csv"), df.values, fmt="%s,%s,%.4f,%.4f,%.4f,%.4f", header=",".join(df.columns), comments="")


if __name__ == "__main__":
    from pyinstrument import Profiler

    profiler = Profiler()
    profiler.start()

    main()

    profiler.stop()

    profiler.print()
