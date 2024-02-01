# glob the files in results directory
import glob
import json
import statistics
from collections import defaultdict

files = glob.glob("results/run-*.json")

runs = set()
runMap = defaultdict(list)
for f in files:
    fname = f.split("/")[1].split(".")[0]
    run = fname.split("-")[1]
    runs.add(run)
    runMap[run].append(f)

# print(runs)

for run in sorted(runs):

    exitCodes = defaultdict(int)
    means = []

    print(f"Run {run}:")
    print(f"    files: {len(runMap[run])}")

    try:
        ps = json.load(open(f"results/parameters-{run}.json"))
        print(f"    params: {ps}")
    except Exception as e:
        print(e)

    for f in runMap[run]:
        try:
            js = json.load(open(f))
            # print(js)
            for code in js["results"][0]["exit_codes"]:
                exitCodes[code] += 1
            means.append(js["results"][0]["mean"])
        # trunk-ignore(flake8/E722)
        except:
            pass
    print(f"    exitCodes: {dict(exitCodes)}")
    if means:
        print(f"    mean: {statistics.mean(means)}")
