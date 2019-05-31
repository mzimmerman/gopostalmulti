#include <stdio.h>
#include <stdlib.h>
#include <libpostal/libpostal.h>
#include <msgpack.h>

int main(int argc, char **argv) {
    // Setup (only called once at the beginning of your program)
    if (!libpostal_setup() || !libpostal_setup_parser()) {
        exit(EXIT_FAILURE);
    }

    libpostal_address_parser_options_t options;
    options = libpostal_get_address_parser_default_options();
    char *lpbuffer;
    size_t lpbufsize = 1024;

    lpbuffer = (char *)malloc(lpbufsize * sizeof(char));
    if( lpbuffer == NULL)
    {
        perror("Unable to allocate buffer");
        exit(1);
    }

	libpostal_address_parser_response_t *parsed;
        msgpack_sbuffer* mpbuffer = msgpack_sbuffer_new();
        msgpack_packer* pk = msgpack_packer_new(mpbuffer, msgpack_sbuffer_write);

    while (true) {
        if (getline(&lpbuffer,&lpbufsize,stdin) <= 0) exit(0); // EOF
	    parsed = libpostal_parse_address(lpbuffer,options);
        /* creates buffer and serializer instance. */

        msgpack_sbuffer_clear(mpbuffer);

        msgpack_pack_array(pk, parsed->num_components);
    	for (size_t i = 0; i < parsed->num_components; i++) {
            msgpack_pack_array(pk, 2);
            msgpack_pack_str(pk, strlen(parsed->labels[i]));
            msgpack_pack_str_body(pk, parsed->labels[i], strlen(parsed->labels[i]));
            msgpack_pack_str(pk, strlen(parsed->components[i]));
            msgpack_pack_str_body(pk, parsed->components[i], strlen(parsed->components[i]));
    	}

        // Free responses
	    libpostal_address_parser_response_destroy(parsed);
        
        if (fwrite(mpbuffer->data,mpbuffer->size, 1, stdout) != 1) exit(0);
        if (fflush(stdout) != 0) exit(0);
    }
    // msgpack_sbuffer_destroy(mpbuffer);
    // return 0;
}
