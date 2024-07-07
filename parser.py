from datetime import datetime
import json
from typing import Any, List, Literal

def duration_into_units(duration_in_seconds: int, desired_units: Literal["m", "h", "s"]) -> str:
    SECONDS_IN_AN_HOUR = 3600
    SECONDS_IN_A_MINUTE = 60

    if desired_units == "m":
        val = duration_in_seconds/SECONDS_IN_A_MINUTE
        val = int(val) if duration_in_seconds % SECONDS_IN_A_MINUTE == 0 else round(val, 2)
        return f'{val}m'
    if desired_units == "h":
        val = duration_in_seconds/SECONDS_IN_AN_HOUR
        val = int(val) if duration_in_seconds % SECONDS_IN_AN_HOUR == 0 else round(val, 2)
        return f'{val}h'
    return str(duration_in_seconds)+"s"

def get_date_key(timestamp: int, units: Literal["d", "w", "m", "y", "h"]):
    start_time = datetime.fromtimestamp(timestamp)
    if units == "m":
        return start_time.strftime("%Y-%m")
    if units == "w":
        return start_time.strftime("%Y-%W")
    if units == "y":
        return start_time.strftime("%Y")
    if units == "d":
        return start_time.strftime("%Y-%m-%d")
    return start_time.strftime("%Y-%m-%d-%H")

def date_key_to_timestamp(date_key: str, units: Literal["d", "w", "m", "y", "h"]):
    if units == "m":
        return date_key
    if units == "w":
        return date_key
    if units == "y":
        return date_key
    if units == "d":
        return date_key
    return datetime.strptime(date_key, "%Y-%m-%d-%H")
   
def group_report(data: List[Any], grouping_units: Literal["d", "w", "m", "y", "h"], units: Literal["m", "h", "s"]):
    
    keys = [(get_date_key(row["start_time"], grouping_units), row["tag"], row["duration"]) for row in data]
    keys = sorted(keys, key=lambda x: (x[0], x[1]))

    result = []
    for row in keys:
        if len(result) < 1:
            result.append(list(row))
            continue
        if row[1] == result[-1][1] and row[0] == result[-1][0]:
            result[-1][2] += row[2]
            continue
        if row[1] != result[-1][1] or row[0] != result[-1][0]:
            result.append(list(row))
            continue
    
    for row in result:
        duration = duration_into_units(row[2], units)
        timestamp = date_key_to_timestamp(row[0], grouping_units)
        print(f'{timestamp}\t{row[1]}\t{duration}')


def report(data: List[Any],
           begin: int | None = None, # inclusive
           end: int | None = None, # exclusive
           tag: str | None = None,
           grouping: Literal["d", "w", "m", "y", "h"] | None = None,
           units: Literal["m", "h", "s"] = "s"
        ):

    data = sorted([row for row in data], key=lambda x: (x["start_time"], x["tag"]))
    # filter on tag
    if tag is not None:
        data = [row for row in data if row["tag"] == tag]

    # filter on begin_date
    if begin is not None:
        data = [row for row in data if row["start_time"] >= begin]

    # filter on end
    if end is not None:
        data = [row for row in data if row["last_timestamp"] < end]

    # group data
    if grouping is not None:
        return group_report(data, grouping, units)
    
    print("start\t\t\tend\t\t\tduration\ttag")
    for row in data:
        duration = duration_into_units(row["duration"], units)
        start_time = datetime.fromtimestamp(row["start_time"])
        end_time = datetime.fromtimestamp(row["last_timestamp"])
        print(f'{start_time}\t{end_time}\t{duration}\t\t{row["tag"]}')

def parse_row(row: str, i: int):
    row_split = row.strip().split(" ")
    state = row_split[0]
    if state not in ["i", "o", "p", "u"]:
        raise Exception(f"Invalid timeclock: Row {i}: {state} should be one of i, o, p, u")
    parsed_time = int(datetime.strptime(" ".join(row_split[1:3]), "%Y-%m-%d %H:%M:%S").timestamp())

    tag_and_notes = " ".join(row_split[3:]).split("  ")
    if len(tag_and_notes) < 2:
        tag_and_notes.append("")
    tag, notes = tag_and_notes
    return parsed_time, state, tag, notes

chars = {
    "i": 1,
    "p": 2,
    "u": 3,
    "o": 0,
}

table = [
    [-1,  1, -1, -1 ], # no timer
    [ 0, -1,  2, -1 ], # running timer
    [ 0,  1, -1,  3 ], # paused timer
    [ 0,  1,  2, -1 ]  # unpaused timer
]
state = [0, 0]
stack = []

if __name__ == "__main__":
    with open("/tmp/t.timeclock", 'r') as f:
        rows = f.readlines()
        for i, row in enumerate(rows):
            [parsed_t, char, tag, notes] = parse_row(row, i)
            state[0], state[1] = state[1], table[state[1]][chars[char]]

            if state[1] == -1:
                raise Exception(f"Row {i}: ({row.strip()}), ERROR: Wasn't expecting \"{char}\".")

            if state == [0, 1]:
                stack.append({
                    "start_time": parsed_t,
                    "last_timestamp": parsed_t,
                    "tag": tag,
                    "duration": 0,
                    "notes": notes,
                })

            if state[0] == 1:
                stack[-1]["duration"] += parsed_t - stack[-1]['last_timestamp']

            stack[-1]["last_timestamp"] = parsed_t

    report(stack)
