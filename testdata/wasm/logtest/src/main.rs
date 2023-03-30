use std::{env, thread, time};
use std::process;
use std::io::{BufRead, BufReader};
use std::fs::File;
use std::error::Error;

const COLOR_RESET: &str = "\x1B[0m";
const COLOR_RED: &str = "\x1B[31m";
const COLOR_GREEN: &str = "\x1B[32m";


// No rand()/srand() in std, and rather than add a dependency that may 
// possibly be problematic this just adds our own random number generator
// using https://en.wikipedia.org/wiki/Linear_congruential_generator. 
// 
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


fn logtest(path: &String) -> Result<(), Box<dyn Error>> {
    let file = File::open(path)?;

    let mut lcg = LCG::new(1234);

    BufReader::new(file).lines().map(
        |line| 
        line.unwrap() 
    ).for_each(|line|  {
        if line.len() % 2 == 0 {
            println!("{}{}",COLOR_GREEN, line);
        } else {
            eprintln!("{}{}",COLOR_RED, line);
        };

        let duration = time::Duration::from_millis(lcg.next(400));        
        thread::sleep(duration);
    });

    Ok(())
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() != 2 {
        let default = String::from("logtest");
        let program_name = args.first().unwrap_or(&default);
        eprintln!("Usage: {} input-txt", program_name);
        process::exit(1);
    } 

    let input_path = &args[1];
    if let Err(err) = logtest(input_path) {
        eprintln!("error: {}", err);
        process::exit(1);
    }

    println!("{}", COLOR_RESET);

    process::exit(0)}
