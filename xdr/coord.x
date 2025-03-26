program Coordinator {
  version Coordinator1 {

    void null(void) = 0;

    // intro_db is called by db shards to introduce themselves to the
    // coordinator.
    IntroDbResult intro_db(DbCard) = 1;

    // intro_executor is called by executors to introduce themselves to the
    // coordinator.
    IntroExecutorResult intro_executor(ExecutorCard) = 2;

    // list_db is called by executors to learn the available set of db shards
    // and whether the db system is ready to use.
    ListDbResult list_db(void) = 3;
  }
  = 1;
}
= 0x3eef0101;

struct IntroDbResult {
  // cc holds the identity of the coordinator that responded
  CoordinatorCard cc;
};

struct IntroExecutorResult {
  // cc holds the identity of the coordinator that responded
  CoordinatorCard cc;
};

struct ListDbResult {
  // cc holds the identity of the coordinator that responded
  CoordinatorCard cc;

  // readiness indicates whether the db cluster is ready to use
  DbReadiness readiness;

  // shards describes all currently available db shards
  DbCard shards<>;
};

enum DbReadiness { NOT_READY = 0, READY = 1 };

// CoordinatorCard uniquely describes a coordinator instance.
struct CoordinatorCard {

  UPID upid;
};

// DbCard uniquely describes a database shard instance and its initial state.
struct DbCard {

  UPID upid;

  unsigned int block_height;

  ServerAddress address;

  // TODO: KeyPrefixes describes the key prefixes handled by this shard
  // KeyPrefixes key_prefixes;
};

// ExecutorCard uniquely describes an executor instance.
struct ExecutorCard {

  UPID upid;

  ServerAddress address;
};

// ServerAddress is an IP (or hostname) and port address, separated by a colon.
typedef string ServerAddress<>;

// UPID is a globally unique process identifier (function of start time, place
// and randomness)
typedef opaque UPID[8];
