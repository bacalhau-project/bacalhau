# Scripting Bacalhau with Python


[![stars - badge-generator](https://img.shields.io/github/stars/bacalhau-project/bacalhau?style=social)](https://github.com/bacalhau-project/bacalhau)

Bacalhau allows you to easily execute batch jobs via the CLI. But sometimes you need to do more than that. You might need to execute a script that requires user input, or you might need to execute a script that requires a lot of parameters. In any case, you probably want to execute your jobs in a repeatable manner.

This example demonstrates a simple Python script that is able to orchestrate the execution of lots of jobs in a repeatable manner.

## TD;LR
Running Python script in Bacalhau

## Prerequisite

To get started, you need to install the Bacalhau client, see more information [here](https://docs.bacalhau.org/getting-started/installation)


## Executing Bacalhau Jobs with Python Scripts

To demonstrate this example, I will use the data generated from the [ethereum analysis example](../../data-engineering/blockchain-etl/index.md). This produced a list of hashes that I will iterate over and execute a job for each one.


```python
%%writefile hashes.txt
bafybeihvtzberlxrsz4lvzrzvpbanujmab3hr5okhxtbgv2zvonqos2l3i
bafybeifb25fgxrzu45lsc47gldttomycqcsao22xa2gtk2ijbsa5muzegq
bafybeig4wwwhs63ly6wbehwd7tydjjtnw425yvi2tlzt3aii3pfcj6hvoq
bafybeievpb5q372q3w5fsezflij3wlpx6thdliz5xowimunoqushn3cwka
bafybeih6te26iwf5kzzby2wqp67m7a5pmwilwzaciii3zipvhy64utikre
bafybeicjd4545xph6rcyoc74wvzxyaz2vftapap64iqsp5ky6nz3f5yndm
```

Now let's run the following script. You can execute this script anywhere with `python bacalhau.py`.


```python
%%writefile bacalhau.py
import json, glob, os, multiprocessing, shutil, subprocess, tempfile, time

# checkStatusOfJob checks the status of a Bacalhau job
def checkStatusOfJob(job_id: str) -> str:
    assert len(job_id) > 0
    p = subprocess.run(
        ["bacalhau", "list", "--output", "json", "--id-filter", job_id],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    r = parseJobStatus(p.stdout)
    if r == "":
        print("job status is empty! %s" % job_id)
    elif r == "Completed":
        print("job completed: %s" % job_id)
    else:
        print("job not completed: %s - %s" % (job_id, r))

    return r


# submitJob submits a job to the Bacalhau network
def submitJob(cid: str) -> str:
    assert len(cid) > 0
    p = subprocess.run(
        [
            "bacalhau",
            "docker",
            "run",
            "--id-only",
            "--wait=false",
            "--input",
            "ipfs://" + cid + ":/inputs/data.tar.gz",
            "ghcr.io/bacalhau-project/examples/blockchain-etl:0.0.6",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if p.returncode != 0:
        print("failed (%d) job: %s" % (p.returncode, p.stdout))
    job_id = p.stdout.strip()
    print("job submitted: %s" % job_id)

    return job_id


# getResultsFromJob gets the results from a Bacalhau job
def getResultsFromJob(job_id: str) -> str:
    assert len(job_id) > 0
    temp_dir = tempfile.mkdtemp()
    print("getting results for job: %s" % job_id)
    for i in range(0, 5): # try 5 times
        p = subprocess.run(
            [
                "bacalhau",
                "get",
                "--output-dir",
                temp_dir,
                job_id,
            ],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        if p.returncode == 0:
            break
        else:
            print("failed (exit %d) to get job: %s" % (p.returncode, p.stdout))

    return temp_dir


# parseJobStatus parses the status of a Bacalhau job
def parseJobStatus(result: str) -> str:
    if len(result) == 0:
        return ""
    r = json.loads(result)
    if len(r) > 0:
        return r[0]["State"]["State"]
    return ""


# parseHashes splits lines from a text file into a list
def parseHashes(filename: str) -> list:
    assert os.path.exists(filename)
    with open(filename, "r") as f:
        hashes = f.read().splitlines()
    return hashes


def main(file: str, num_files: int = -1):
    # Use multiprocessing to work in parallel
    count = multiprocessing.cpu_count()
    with multiprocessing.Pool(processes=count) as pool:
        hashes = parseHashes(file)[:num_files]
        print("submitting %d jobs" % len(hashes))
        job_ids = pool.map(submitJob, hashes)
        assert len(job_ids) == len(hashes)

        print("waiting for jobs to complete...")
        while True:
            job_statuses = pool.map(checkStatusOfJob, job_ids)
            total_finished = sum(map(lambda x: x == "Completed", job_statuses))
            if total_finished >= len(job_ids):
                break
            print("%d/%d jobs completed" % (total_finished, len(job_ids)))
            time.sleep(2)

        print("all jobs completed, saving results...")
        results = pool.map(getResultsFromJob, job_ids)
        print("finished saving results")

        # Do something with the results
        shutil.rmtree("results", ignore_errors=True)
        os.makedirs("results", exist_ok=True)
        for r in results:
            path = os.path.join(r, "outputs", "*.csv")
            csv_file = glob.glob(path)
            for f in csv_file:
                print("moving %s to results" % f)
                shutil.move(f, "results")

if __name__ == "__main__":
    main("hashes.txt", 10)

```

This code has a few interesting features:
* Change the value in the `main` call to change the number of jobs to execute
* Because all jobs are complete at different times, there's a loop to check that all jobs have been completed before downloading the results -- if you don't do this you'll likely see an error when trying to download the results
* When downloading the results, the IPFS get often times out, so I wrapped that in a loop

Let's run it!


```bash
%%bash
python bacalhau.py
```

Hopefully, the results directory contains all the combined results from the jobs we just executed. Here's we're expecting to see CSV files:


```bash
%%bash
ls -l results
```

Success! We've now executed a bunch of jobs in parallel using Python. This is a great way to execute lots of jobs in a repeatable manner. You can alter the file above for your purposes.

### Next Steps

You might also be interested in the following examples:

* [Analysing Ethereum Data with Python](../../data-engineering/blockchain-etl/index.md)
