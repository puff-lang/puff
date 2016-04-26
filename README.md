# puff

Purely functional language that feels familiar to JavaScript developers.

## Usage

To compile:

    make

To enter REPL:

    ./puff

To run a `.puff` file (using builtin evaluator):

    ./puff run examples/factorial.puff

To compile a `.puff` file to an executable:

    ./puff build examples/factorial.puff

## Examples

factorial.puff

    fn fact(n) => if n == 0 then 1 else n * fact(n - 1)
    fn main() => fact(5)

gcd.puff

    fn gcd(a, b) => if b == 0 then a else gcd(b, (a % b))
    fn main() => gcd(49, 35)

## License
MIT