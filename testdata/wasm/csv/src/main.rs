use std::fs::File;
use std::error::Error;
use std::process;

const HORSE_ID: usize = 0;
const HORSE_LABEL: usize = 1;
const MOTHER_ID: usize = 2;
const MOTHER_LABEL: usize = 3;
const FATHER_ID : usize = 4;
const FATHER_LABEL: usize = 5;

fn run() -> Result<(), Box<dyn Error>> {
    // Build the CSV reader and iterate over each record.
    let input = File::open("inputs/horses.csv")?;
    let mut wtr = csv::Writer::from_path("outputs/parents-children.csv")?;
    let mut rdr = csv::Reader::from_reader(input);
    
    let headers = csv::ByteRecord::from(vec!["parent", "parentLabel", "child", "childLabel"]);
    wtr.write_byte_record(&headers)?;
    
    for result in rdr.records() {
        let record = result?;
        let horse_id = record.get(HORSE_ID).or(Some("")).unwrap();
        let horse_label = record.get(HORSE_LABEL).or(Some("")).unwrap();
        let mother_id = record.get(MOTHER_ID).or(Some("")).unwrap();
        let mother_label = record.get(MOTHER_LABEL).or(Some("")).unwrap();
        let father_id = record.get(FATHER_ID).or(Some("")).unwrap();
        let father_label = record.get(FATHER_LABEL).or(Some("")).unwrap();

        if mother_id != "" {
            let mother_record = csv::StringRecord::from(vec![mother_id, mother_label, horse_id, horse_label]);
            wtr.write_byte_record(&mother_record.into_byte_record())?;
        }

        if father_id != "" {
            let father_record = csv::StringRecord::from(vec![father_id, father_label, horse_id, horse_label]);
            wtr.write_byte_record(&father_record.into_byte_record())?;
        }
    }
    
    Ok(())
}

fn main() {
    if let Err(err) = run() {
        eprintln!("error: {}", err);
        process::exit(1);
    }

    process::exit(0)
}