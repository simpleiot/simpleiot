module Components.NodeProvisioning exposing (view)

import Api.Node as Node
import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import UI.Icon as Icon
import UI.Style exposing (colors)


{-| The provisioning node gathers the files this instance is configured from:
provisioningFile nodes recording what was applied from the provisioning
directory, and file nodes uploaded through this UI.
-}
view : NodeOptions msg -> Element msg
view o =
    let
        sources =
            List.filter isSource o.children

        failed =
            List.filter hasError sources
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.list
            , text "Provisioning"
            , text <| summary (List.length sources)
            , viewFailed (List.length failed)
            ]
            :: (if o.expDetail then
                    [ paragraph [ paddingEach { top = 0, bottom = 0, left = 70, right = 0 } ]
                        [ text <|
                            "Files applied to this instance. Add a file here to configure it "
                                ++ "the way a file in the provisioning directory does."
                        ]
                    ]

                else
                    []
               )


{-| isSource picks out the children that describe a provisioning file, so that
anything else living under this node is not counted.
-}
isSource : { a | node : { b | typ : String } } -> Bool
isSource c =
    c.node.typ == Node.typeProvisioningFile || c.node.typ == Node.typeFile


hasError : { a | node : { b | points : List Point.Point } } -> Bool
hasError c =
    Point.getText c.node.points Point.typeError "" /= ""


summary : Int -> String
summary count =
    case count of
        0 ->
            "no files"

        1 ->
            "1 file"

        _ ->
            String.fromInt count ++ " files"


{-| viewFailed makes a file that did not apply visible without expanding
anything, which is the question this node exists to answer.
-}
viewFailed : Int -> Element msg
viewFailed count =
    if count == 0 then
        none

    else
        el [ Font.color colors.red ] <|
            text <|
                if count == 1 then
                    "1 with an error"

                else
                    String.fromInt count ++ " with errors"
