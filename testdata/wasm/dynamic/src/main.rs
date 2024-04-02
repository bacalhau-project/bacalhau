#[link(wasm_import_module="QmPympgyrEGEdSJ93rqvQkR71QLuQGdhKQtYztFwxpQsid")]
extern "C" {
    pub fn easter(year: i32) -> i32;
}

fn main() {
    unsafe {
        println!("{}", easter(2022))
    }
}
