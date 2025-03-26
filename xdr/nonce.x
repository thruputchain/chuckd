
// TCP port to use
const XDRDEMO_PORT = 30428;

typedef string Key<64>;
typedef string Value<512>;

enum Status {
  SUCCESS = 0,			// Operatgion succeeded
  NOTFOUND = 1,			// Key was not found
  SYSTEMERR = 2			// Some kind of system error occurred
};

union GetRes switch (Status stat) {
  case SUCCESS:
    Value value;
  default:
    void;
};

program KVPROG {
  version KVPROT1 {
    void kv_null(void) = 0;
    Status kv_put(Key, Value) = 1;
    GetRes kv_get(Key) = 2;
  } = 1;
} = 0x2b0eefac;
