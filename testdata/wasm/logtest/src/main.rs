use std::error::Error;
use std::fs;
use std::io::{BufRead, BufReader};
use std::process;
use std::{env, thread, time};

const COLOR_RESET: &str = "\x1B[0m";
const COLOR_RED: &str = "\x1B[31m";
const COLOR_GREEN: &str = "\x1B[32m";

// No rand()/srand() in std, and rather than add a dependency that may
// possibly be problematic this just adds our own random number generator
// using https://en.wikipedia.org/wiki/Linear_congruential_generator.
struct LCG {
    seed: u64,
    a: u64,
    c: u64,
    modulous: u64,
}

impl LCG {
    // New LCG using the Visual Basic starting params
    fn new(seed: u64) -> Self {
        LCG {
            seed,
            a: 1140671485,
            c: 12820163,
            modulous: 16777216,
        }
    }

    fn next(&mut self, max: u64) -> u64 {
        self.seed = (self.a.wrapping_mul(self.seed) + self.c) % self.modulous;
        self.seed.clone() & max
    }
}

fn logtest(path: &String, pauser: Box<dyn Fn(&mut LCG)>) -> Result<(), Box<dyn Error>> {
    let file = match fs::File::open(path) {
        Ok(f) => f, 
        Err(e) => {
            eprintln!("failed to open file : {path}");
            return Err(Box::new(e)) // Return the error after reboxing it
        }
    };

    let mut lcg = LCG::new(&file as *const _ as u64);

    BufReader::new(file)
        .lines()
        .map(|line| line.unwrap())
        .for_each(|line| {
            if line.len() % 2 == 0 {
                println!("{}{}", COLOR_GREEN, line);
            } else {
                eprintln!("{}{}", COLOR_RED, line);
            };

            pauser(&mut lcg);
        });

    Ok(())
}

fn main() {
    let args: Vec<String> = env::args().collect();

    let slow: bool;
    let file: &String;

    (file, slow) = if let [_program, filename, slowflag] = &args[..] {
        (filename, slowflag == "--slow")
    } else if let [_program, filename] = &args[..] {
        (filename, false)
    } else {
        eprintln!("Usage: logtest input-txt [--slow]");
        process::exit(1);
    };

    // Create a closure that will either do nothing, or if we specify
    // --slow then will pause for up to 400ms between lines.
    let mut pauser: Box<dyn Fn(&mut LCG)> = Box::new(|_lcg: &mut LCG| {});
    if slow {
        pauser = Box::new(|lcg: &mut LCG| {
            let millis = lcg.next(400);
            let duration = time::Duration::from_millis(millis);
            thread::sleep(duration);
        });
    }

    match logtest(&file, pauser) {
        Err(err) => {
            eprintln!("Error: {err:?} : failed to open {file}");
            process::exit(2);
        }
        Ok(()) => {
            println!("{}", COLOR_RESET);
        }
    }

    process::exit(0)
}
