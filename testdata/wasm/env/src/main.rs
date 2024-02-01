use std::{env, collections::BTreeMap};

fn main() {
    let mut map: BTreeMap<String, String> = BTreeMap::new();
    map.extend(env::vars().into_iter());
    map.iter().for_each(|(k, v)| println!("{k}={v}") )
}
