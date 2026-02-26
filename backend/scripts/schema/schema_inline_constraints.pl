use strict;
use warnings;

local $/;
my $input = <STDIN>;

sub normalize_name {
    my ($name) = @_;
    $name =~ s/^"//;
    $name =~ s/"$//;
    return lc $name;
}

sub inline_constraints {
    my ($block) = @_;
    my @lines = split /\n/, $block;
    my $header = shift @lines;
    my $footer = pop @lines;

    my @columns;
    my @constraints;
    for my $line (@lines) {
        next if $line =~ /^\s*$/;
        if ( $line =~ /^\s*constraint\s+/i ) {
            push @constraints, $line;
        } else {
            push @columns, $line;
        }
    }

    my %col_order;
    my @col_defs;
    for my $line (@columns) {
        $line =~ s/,\s*$//;
        my ($name) = $line =~ /^\s*("?[a-zA-Z0-9_]+"?)\s+/;
        next unless defined $name;
        my $norm = normalize_name($name);
        $col_order{$norm} = {
            line   => $line,
            inline => [],
        };
        push @col_defs, $norm;
    }

    my @remaining_constraints;
    for my $line (@constraints) {
        $line =~ s/,\s*$//;
        $line =~ s/^\s*//;

        if ( $line =~ /^constraint\s+(\S+)\s+primary\s+key\s*\(([^)]+)\)$/i ) {
            my ($name, $cols) = ($1, $2);
            if ( $cols !~ /,/ ) {
                my $norm = normalize_name($cols);
                if ( $col_order{$norm} ) {
                    push @{ $col_order{$norm}->{inline} }, "constraint $name primary key";
                    next;
                }
            }
        } elsif ( $line =~ /^constraint\s+(\S+)\s+unique\s*\(([^)]+)\)$/i ) {
            my ($name, $cols) = ($1, $2);
            if ( $cols !~ /,/ ) {
                my $norm = normalize_name($cols);
                if ( $col_order{$norm} ) {
                    push @{ $col_order{$norm}->{inline} }, "constraint $name unique";
                    next;
                }
            }
        } elsif ( $line =~ /^constraint\s+(\S+)\s+foreign\s+key\s*\(([^)]+)\)\s+references\s+(.+)$/i ) {
            my ($name, $cols, $ref) = ($1, $2, $3);
            if ( $cols !~ /,/ ) {
                my $norm = normalize_name($cols);
                if ( $col_order{$norm} ) {
                    push @{ $col_order{$norm}->{inline} }, "constraint $name references $ref";
                    next;
                }
            }
        }

        push @remaining_constraints, $line;
    }

    my @items;
    for my $norm (@col_defs) {
        my $entry = $col_order{$norm};
        my $line = $entry->{line};
        if ( @{ $entry->{inline} } ) {
            $line .= ' ' . join(' ', @{ $entry->{inline} });
        }
        push @items, $line =~ /^\s*/ ? $line : "  $line";
    }

    for my $line (@remaining_constraints) {
        push @items, $line =~ /^\s*/ ? "  $line" : "  $line";
    }

    return join("\n", $header, join(",\n", @items), $footer, '');
}

$input =~ s/(create table [\s\S]*?\n\);)/inline_constraints($1)/gei;
$input =~ s/\n{3,}/\n\n/g;

print $input;
