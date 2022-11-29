# Scripting Bacalhau with Python

[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/bacalhau-project/examples/blob/main/workload-onboarding/python-script/index.ipynb)
[![Open In Binder](https://mybinder.org/badge.svg)](https://mybinder.org/v2/gh/bacalhau-project/examples/HEAD?labpath=workload-onboarding/python-script/index.ipynb)

Bacalhau allows you to easily execute batch jobs via the CLI. But sometimes you need to do more than that. You might need to execute a script that requires user input, or you might need to execute a script that requires a lot of parameters. In any case, you probably want to execute your jobs in a repeatable manner.

This example demonstrates a simple Python script that is able to orchestrate the execution of lots of jobs in a repeatable manner.

### Prerequisites

* Python
* The Bacalhau client - [Installation instructions](https://docs.bacalhau.org/getting-started/installation)


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
bafybeicgo3iofo3sw73wenc3nkdhi263yytjnds5cxjwvypwekbz4sk7ra
bafybeihvep5xsvxm44lngmmeysihsopcuvcr34an4idz45ixl5slsqzy3y
bafybeigmt2zwzrbzwb4q2kt2ihlv34ntjjwujftvabrftyccwzwdypama4
bafybeiciwui7sw3zqkvp4d55p4woq4xgjlstrp3mzxl66ab5ih5vmeozci
bafybeicpmotdsj2ambf666b2jkzp2gvg6tadr6acxqw2tmdlmsruuggbbu
bafybeigefo3esovbveavllgv5wiheu5w6cnfo72jxe6vmfweco5eq5sfty
bafybeigvajsumnfwuv7lp7yhr2sr5vrk3bmmuhhnaz53waa2jqv3kgkvsu
bafybeih2xg2n7ytlunvqxwqlqo5l3daykuykyvhgehoa2arot6dmorstmq
bafybeihnmq2ltuolnlthb757teihwvvw7wophoag2ihnva43afbeqdtgi4
bafybeibb34hzu6z2xgo6nhrplt3xntpnucthqlawe3pmzgxccppbxrpudy
bafybeigny33b4g6gf2hrqzzkfbroprqrimjl5gmb3mnsqu655pbbny6tou
bafybeifgqjvmzbtz427bne7af5tbndmvniabaex77us6l637gqtb2iwlwq
bafybeibryqj62l45pxjhdyvgdc44p3suhvt4xdqc5jpx474gpykxwgnw2e
bafybeidme3fkigdjaifkjfbwn76jk3fcqdogpzebtotce6ygphlujaecla
bafybeig7myc3eg3h2g5mk2co7ybte4qsuremflrjneer6xk3pghjwmcwbi
bafybeic3x2r5rrd3fdpdqeqax4bszcciwepvbpjl7xdv6mkwubyqizw5te
bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq
bafybeicumr67jkyarg5lspqi2w4zqopvgii5dgdbe5vtbbq53mbyftduxy
bafybeiecn2cdvefvdlczhz6i4afbkabf5pe5yqrcsgdvlw5smme2tw7em4
bafybeiaxh7dhg4krgkil5wqrv5kdsc3oewwy6ym4n3545ipmzqmxaxrqf4
bafybeiclcqfzinrmo3adr4lg7sf255faioxjfsolcdko3i4x7opx7xrqii
bafybeicjmeul7c2dxhmaudawum4ziwfgfkvbgthgtliggfut5tsc77dx7q
bafybeialziupik7csmhfxnhuss5vrw37kmte7rmboqovp4cpq5hj4insda
bafybeid7ecwdrw7pb3fnkokq5adybum6s5ok3yi2lw4m3edjpuy65zm4ji
bafybeibuxwnl5ogs4pwa32xriqhch24zbrw44rp22hrly4t6roh6rz7j4m
bafybeicxvy47jpvv3fi5umjatem5pxabfrbkzxiho7efu6mpidjpatte54
bafybeifynb4mpqrbsjbeqtxpbuf6y4frrtjrc4tm7cnmmui7gbjkckszrq
bafybeidcgnbhguyfaahkoqbyy2z525d3qfzdtbjuk4e75wkdbnkcafvjei
bafybeiefc67s6hpydnsqdgypbunroqwkij5j26sfmc7are7yxvg45uuh7i
bafybeiefwjy3o42ovkssnm7iihbog46k5grk3gobvvkzrqvof7p6xbgowi
bafybeihpydd3ivtza2ql5clatm5fy7ocych7t4czu46sbc6c2ykrbwk5uu
bafybeiet7222lqfmzogur3zlxqavlnd3lt3qryw5yi5rhuiqeqg4w7c3qu
bafybeihwomd4ygoydvj5kh24wfwk5kszmst5vz44zkl6yibjargttv7sly
bafybeidbjt2ckr4oooio3jsfk76r3bsaza5trjvt7u36slhha5ksoc5gv4
bafybeifyjrmopgtfmswq7b4pfscni46doy3g3z6vi5rrgpozc6duebpmuy
bafybeidsrowz46yt62zs64q2mhirlc3rsmctmi3tluorsts53vppdqjj7e
bafybeiggntql57bw24bw6hkp2yqd3qlyp5oxowo6q26wsshxopfdnzsxhq
bafybeidguz36u6wakx4e5ewuhslsfsjmk5eff5q7un2vpkrcu7cg5aaqf4
bafybeiaypwu2b45iunbqnfk2g7bku3nfqveuqp4vlmmwj7o7liyys42uai
bafybeicaahv7xvia7xojgiecljo2ddrvryzh2af7rb3qqbg5a257da5p2y
bafybeibgeiijr74rcliwal3e7tujybigzqr6jmtchqrcjdo75trm2ptb4e
bafybeiba3nrd43ylnedipuq2uoowd4blghpw2z7r4agondfinladcsxlku
bafybeif3semzitjbxg5lzwmnjmlsrvc7y5htekwqtnhmfi4wxywtj5lgoe
bafybeiedmsig5uj7rgarsjans2ad5kcb4w4g5iurbryqn62jy5qap4qq2a
bafybeidyz34bcd3k6nxl7jbjjgceg5eu3szbrbgusnyn7vfl7facpecsce
bafybeigmq5gch72q3qpk4nipssh7g7msk6jpzns2d6xmpusahkt2lu5m4y
bafybeicjzoypdmmdt6k54wzotr5xhpzwbgd3c4oqg6mj4qukgvxvdrvzye
bafybeien55egngdpfvrsxr2jmkewdyha72ju7qaaeiydz2f5rny7drgzta
```

    Overwriting hashes.txt


Now let's run the following script. There's a fair bit of code, but basically there's three core functions to submit, check the status of, and download the results from, a job. Then the main function wraps all of that in a `multiprocessing` pool to execute the jobs in parallel. Feel free to copy this code and save it to your local machine as `bacalhau.py`. Then you can execute this script anywhere with `python bacalhau.py`.


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
            "--input-volumes",
            cid + ":/inputs/data.tar.gz",
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
        for _, v in r[0]["JobState"]["Nodes"].items():
            state = v["Shards"]["0"]["State"]
            if state == "Completed":
                return state
        for _, v in r[0]["JobState"]["Nodes"].items():
            state = v["Shards"]["0"]["State"]
            if state != "Cancelled":
                return state
        return "Error"
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
        shutil.rmtree("results")
        os.makedirs("results", exist_ok=True)
        for r in results:
            path = os.path.join(r, "combined_results", "outputs", "*.csv")
            csv_file = glob.glob(path)
            for f in csv_file:
                print("moving %s to results" % f)
                shutil.move(f, "results")

if __name__ == "__main__":
    main("hashes.txt", 10)

```

This code has a few interesting features:
* Change the value in the `main` call to change the number of jobs to execute
* Because all jobs complete at different times, there's a loop to check that all jobs have completed before downloading the results -- if you don't do this you'll likely see an error when trying to download the results
* When downloading the results, the IPFS get often times out, so I wrapped that in a loop

Let's run it!


```bash
python bacalhau.py
```

    submitting 10 jobs
    job submitted: 239f89b9-c9c2-4b24-b314-030f20d2ed86
    job submitted: 5ce48688-fe02-4514-a9d0-a16f43634e8a
    job submitted: 1ab7f06c-4c50-4617-8994-1904aa3bb1de
    job submitted: 04201261-467b-4985-9b25-dacd59a2b6ab
    job submitted: c2d181fd-ec3a-4320-abe2-01dd66784b5f
    job submitted: 99a1d838-4ecc-45aa-ba02-67afcf4eaa4c
    job submitted: 8d1bfc74-d623-45af-866d-d8d3100889e0
    job submitted: ff777c33-04c5-43ca-a291-397c6e11e297
    job submitted: 8cbfffa4-aef7-4413-9042-6776d5eddc33
    job submitted: 50f0a7f1-6619-4b7b-ad53-65858389a8cd
    waiting for jobs to complete...
    job not completed: 8d1bfc74-d623-45af-866d-d8d3100889e0 - Waiting
    job not completed: 1ab7f06c-4c50-4617-8994-1904aa3bb1de - Waiting
    job not completed: 5ce48688-fe02-4514-a9d0-a16f43634e8a - Waiting
    job not completed: 04201261-467b-4985-9b25-dacd59a2b6ab - Waiting
    job not completed: ff777c33-04c5-43ca-a291-397c6e11e297 - Waiting
    job not completed: 99a1d838-4ecc-45aa-ba02-67afcf4eaa4c - Waiting
    job not completed: c2d181fd-ec3a-4320-abe2-01dd66784b5f - Waiting
    job not completed: 239f89b9-c9c2-4b24-b314-030f20d2ed86 - Waiting
    job not completed: 50f0a7f1-6619-4b7b-ad53-65858389a8cd - Waiting
    job not completed: 8cbfffa4-aef7-4413-9042-6776d5eddc33 - Waiting
    0/10 jobs completed
    job not completed: ff777c33-04c5-43ca-a291-397c6e11e297 - Waiting
    job not completed: 04201261-467b-4985-9b25-dacd59a2b6ab - Waiting
    job completed: 239f89b9-c9c2-4b24-b314-030f20d2ed86
    job not completed: 1ab7f06c-4c50-4617-8994-1904aa3bb1de - Waiting
    job not completed: c2d181fd-ec3a-4320-abe2-01dd66784b5f - Waiting
    job not completed: 8d1bfc74-d623-45af-866d-d8d3100889e0 - Waiting
    job completed: 99a1d838-4ecc-45aa-ba02-67afcf4eaa4c
    job completed: 5ce48688-fe02-4514-a9d0-a16f43634e8a
    job not completed: 50f0a7f1-6619-4b7b-ad53-65858389a8cd - Waiting
    job not completed: 8cbfffa4-aef7-4413-9042-6776d5eddc33 - Waiting
    3/10 jobs completed
    job completed: 5ce48688-fe02-4514-a9d0-a16f43634e8a
    job completed: c2d181fd-ec3a-4320-abe2-01dd66784b5f
    job completed: 04201261-467b-4985-9b25-dacd59a2b6ab
    job completed: 8d1bfc74-d623-45af-866d-d8d3100889e0
    job completed: ff777c33-04c5-43ca-a291-397c6e11e297
    job completed: 239f89b9-c9c2-4b24-b314-030f20d2ed86
    job completed: 99a1d838-4ecc-45aa-ba02-67afcf4eaa4c
    job completed: 1ab7f06c-4c50-4617-8994-1904aa3bb1de
    job completed: 8cbfffa4-aef7-4413-9042-6776d5eddc33
    job completed: 50f0a7f1-6619-4b7b-ad53-65858389a8cd
    all jobs completed, saving results...
    getting results for job: c2d181fd-ec3a-4320-abe2-01dd66784b5f
    getting results for job: 239f89b9-c9c2-4b24-b314-030f20d2ed86
    getting results for job: 99a1d838-4ecc-45aa-ba02-67afcf4eaa4c
    getting results for job: ff777c33-04c5-43ca-a291-397c6e11e297
    getting results for job: 5ce48688-fe02-4514-a9d0-a16f43634e8a
    getting results for job: 04201261-467b-4985-9b25-dacd59a2b6ab
    getting results for job: 8d1bfc74-d623-45af-866d-d8d3100889e0
    getting results for job: 1ab7f06c-4c50-4617-8994-1904aa3bb1de
    getting results for job: 50f0a7f1-6619-4b7b-ad53-65858389a8cd
    getting results for job: 8cbfffa4-aef7-4413-9042-6776d5eddc33
    finished saving results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpui2gdnta/combined_results/outputs/transactions_00000000_00049999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmphktn81xx/combined_results/outputs/transactions_00050000_00099999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmp15x2meyr/combined_results/outputs/transactions_00100000_00149999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmp7md5a9yc/combined_results/outputs/transactions_00150000_00199999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpf_3mxm23/combined_results/outputs/transactions_00200000_00249999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpgybx5jqy/combined_results/outputs/transactions_00250000_00299999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpsvdyyig7/combined_results/outputs/transactions_00300000_00349999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpsjyrsujx/combined_results/outputs/transactions_00350000_00399999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmpo76qn5fi/combined_results/outputs/transactions_00400000_00449999.csv to results
    moving /var/folders/kr/pl4p96k11b55hp5_p9l_t8kr0000gn/T/tmp3xoyumpj/combined_results/outputs/transactions_00450000_00499999.csv to results


Hopefully the results directory contains all the combined results from the jobs we just executed. Here's we're expecting to see csv files:


```bash
ls -l results
```

    total 40
    -rw-r--r-- 3 phil staff  55 Nov 29 11:16 transactions_00000000_00049999.csv
    -rw-r--r-- 3 phil staff 387 Nov 29 11:17 transactions_00050000_00099999.csv
    -rw-r--r-- 3 phil staff 388 Nov 29 11:17 transactions_00100000_00149999.csv
    -rw-r--r-- 3 phil staff 426 Nov 29 11:16 transactions_00150000_00199999.csv
    -rw-r--r-- 3 phil staff 393 Nov 29 11:17 transactions_00200000_00249999.csv
    -rw-r--r-- 3 phil staff 384 Nov 29 11:16 transactions_00250000_00299999.csv
    -rw-r--r-- 3 phil staff 421 Nov 29 11:16 transactions_00300000_00349999.csv
    -rw-r--r-- 3 phil staff 390 Nov 29 11:17 transactions_00350000_00399999.csv
    -rw-r--r-- 3 phil staff 347 Nov 29 11:17 transactions_00400000_00449999.csv
    -rw-r--r-- 3 phil staff 386 Nov 29 11:18 transactions_00450000_00499999.csv


Success! We've now executed a bunch of jobs in parallel using Python. This is a great way to execute lots of jobs in a repeatable manner. You can alter the file above for your purposes.

### Next Steps

You might also be interested in the following examples:

* [Analysing Ethereum Data with Python](../../data-engineering/blockchain-etl/index.md)
