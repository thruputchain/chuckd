struct Request {
	opaque payload<>;
};

struct Reply {
	opaque payload<>;
};

program Echo
{
	version Echo1
	{

		void null(void) = 0;

		Reply echo(Request) = 10;

		void no_echo(Request) = 20;
	}
	= 1;
}
= 0x100ff001;
